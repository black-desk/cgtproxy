package core

/*
#cgo pkg-config: libnftables
#include <stdlib.h>
#include <nftables/libnftables.h>
*/
import "C"

import (
	"bytes"
	_ "embed"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
	"unsafe"

	"github.com/black-desk/deepin-network-proxy-manager/internal/log"
	"github.com/google/nftables"
	"github.com/google/nftables/expr"
	"github.com/vishvananda/netlink"
)

const (
	_nftTableName = "deepin_proxy"
	_tproxyFwMark = 0x64707470
	_tproxyTable  = 7470
	_cgroupBase   = "/sys/fs/cgroup" // TODO: detect the cgroup2 mountpoint
)

//go:embed assets/init_table.nft
var _nftInitTplContent string

var _nftInitTpl = template.Must(template.New("init_nft_table").Parse(_nftInitTplContent))

type nftTplData struct {
	TableName    string
	TProxyHost   string
	TProxyPort   uint16
	TProxyFwMark uint32
}

type ruleManager struct {
	EventChan <-chan *cgroupEvent `inject:"true"`

	netlinkConn *nftables.Conn
	nftCtx      *C.struct_nft_ctx
}

func (c *Core) newRuleManager() (m *ruleManager, err error) {
	defer func() {
		if err == nil {
			return
		}

		err = fmt.Errorf(
			"Failed to create the nftable rule manager: %w",
			err,
		)
	}()

	m = &ruleManager{}
	if err = c.container.Fill(m); err != nil {
		return
	}

	if m.netlinkConn, err = nftables.New(); err != nil {
		return
	}

	return
}

func (c *Core) runRuleManager() (err error) {
	var m *ruleManager
	if m, err = c.newRuleManager(); err != nil {
		return
	}

	if m.nftCtx = C.nft_ctx_new(C.NFT_CTX_DEFAULT); m.nftCtx == nil {
		err = fmt.Errorf("cannot allocate nft context")
		return
	}
	defer C.nft_ctx_free(m.nftCtx)

	err = m.run()
	return
}

func (m *ruleManager) run() (err error) {
	defer func() {
		if err == nil {
			return
		}

		err = fmt.Errorf(
			"Error occurs while running the nftable rules manager: %w",
			err,
		)
	}()

	defer m.removeNftableRules()
	if err = m.initializeNftableRuels(); err != nil {
		return
	}

	rule := netlink.NewRule()
	rule.Family = netlink.FAMILY_ALL
	rule.Mark = _tproxyFwMark
	rule.Table = _tproxyTable
	if err = netlink.RuleAdd(rule); err != nil {
		return
	}
	defer func() {
		if err := netlink.RuleDel(rule); err != nil {
			log.Warning().Printf("failed to delete rule: %v", err)
		}
	}()

	// route := new(netlink.Route)
	// route.Table = _tproxyTable
	routeAddCmd := fmt.Sprintf("ip route add default dev lo table %d", _tproxyTable)
	if err = exec.Command("sh", "-c", routeAddCmd).Run(); err != nil {
		return
	}
	defer func() {
		routeDelCmd := fmt.Sprintf("ip route delete default dev lo table %d", _tproxyTable)
		if err := exec.Command("sh", "-c", routeDelCmd).Run(); err != nil {
			log.Warning().Printf("failed to delete route: %v", err)
		}
	}()

	for event := range m.EventChan {
		log.Debug().Printf("handle cgroup event %+v", event)

		// strip '/sys/fs/cgroup'
		cgroupPath, err := filepath.Rel(_cgroupBase, event.Path)
		if err != nil {
			log.Warning().Printf("wrong cgroup path: %s", event.Path)
			continue
		}
		level := len(strings.Split(cgroupPath, "/"))
		cmd := fmt.Sprintf(`add rule inet %s mangle_output socket cgroupv2 level %d "%s" jump tproxy_mark`, _nftTableName, level, cgroupPath)
		if err := m.runNftCmd(cmd); err != nil {
			log.Warning().Print(err)
			continue
		}
	}
	return
}

func (m *ruleManager) initializeNftableRuels() (err error) {
	if m.netlinkConn, err = nftables.New(); err != nil {
		return
	}

	table := &nftables.Table{
		Name:   _nftTableName,
		Family: nftables.TableFamilyINet,
	}
	m.netlinkConn.AddTable(table)

	tproxyMarkChain := &nftables.Chain{
		Table: table,
		Name:  "tproxy_mark",
	}
	m.netlinkConn.AddChain(tproxyMarkChain)

	tproxyMarkRules := &nftables.Rule{
		Table: table,
		Chain: tproxyMarkChain,
		Exprs: []expr.Any{
			&expr.Meta{
				Key: expr.MetaKeyL4PROTO,
			},
		},
	}

	outputChainPolicy := nftables.ChainPolicyAccept
	outputChain := &nftables.Chain{
		Table:    table,
		Name:     "mangle_output",
		Type:     nftables.ChainTypeRoute,
		Hooknum:  nftables.ChainHookOutput,
		Priority: nftables.ChainPriorityMangle,
		Policy:   &outputChainPolicy,
	}
	m.netlinkConn.AddChain(outputChain)

	preroutingChainPolicy := nftables.ChainPolicyAccept
	preroutingChain := &nftables.Chain{
		Table:    table,
		Name:     "mangle_prerouting",
		Type:     nftables.ChainTypeFilter,
		Hooknum:  nftables.ChainHookPrerouting,
		Priority: nftables.ChainPriorityMangle,
		Policy:   &preroutingChainPolicy,
	}
	m.netlinkConn.AddChain(preroutingChain)

	buf := &bytes.Buffer{}
	_nftInitTpl.Execute(buf, nftTplData{
		TableName:    _nftTableName,
		TProxyHost:   "127.0.0.1",
		TProxyPort:   7893,
		TProxyFwMark: _tproxyFwMark,
	})

	err = m.runNftCmd(buf.String())

	return
}

func (m *ruleManager) removeNftableRules() error {
	cmd := fmt.Sprintf("delete table inet %s", _nftTableName)
	return m.runNftCmd(cmd)
}

func (m *ruleManager) runNftCmd(cmd string) error {
	cmdCStr := C.CString(cmd)
	defer C.free(unsafe.Pointer(cmdCStr))

	res := C.nft_run_cmd_from_buffer(m.nftCtx, cmdCStr)
	if res < 0 {
		return fmt.Errorf("failed to add nftable rule(%d): %s", res, cmd)
	}

	return nil
}
