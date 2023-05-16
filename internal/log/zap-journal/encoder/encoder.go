package encoder

import (
	"fmt"
	"log/syslog"

	"github.com/black-desk/deepin-network-proxy-manager/internal/log/zap-journal/internal/bufferpool"
	"github.com/black-desk/deepin-network-proxy-manager/internal/log/zap-journal/internal/zapjson"
	"github.com/black-desk/deepin-network-proxy-manager/internal/pool"
	"github.com/black-desk/deepin-network-proxy-manager/pkg/location"
	"go.uber.org/zap"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

type JournalEncoder struct {
	facility syslog.Priority
	cfg      zapcore.EncoderConfig

	jenc *zapjson.JsonEncoder
	buf  *buffer.Buffer

	lengthPos int
	nss       []string
}

var journalEncoderPool = pool.New(
	func() any {
		return &JournalEncoder{}
	},
	func(x any) any {
		enc := x.(*JournalEncoder)
		if enc.jenc != nil {
			enc.jenc.Free()
		}
		enc.jenc = nil
		if enc.buf != nil {
			enc.buf.Free()
		}
		enc.buf = nil
		enc.nss = nil
		return enc
	},
)

type Opt = (func(*JournalEncoder) (*JournalEncoder, error))

func New(opts ...Opt) (ret *JournalEncoder, err error) {
	defer func() {
		if err == nil {
			return
		}
		err = fmt.Errorf(location.Capture()+
			"Failed to create new journal encoder:\n%w",
			err,
		)
	}()

	enc := journalEncoderPool.Get().(*JournalEncoder)

	enc.facility = syslog.LOG_USER

	for i := range opts {
		enc, err = opts[i](enc)
		if err != nil {
			return
		}
	}

	if enc.buf != nil {
		panic("this should never happened")
	}

	enc.buf = bufferpool.Get()

	if enc.jenc != nil {
		panic("this should never happened")
	}

	enc.jenc = zapjson.New(enc.cfg, enc.buf)

	ret = enc

	return
}

func WithCfg(cfg zapcore.EncoderConfig) Opt {
	return func(enc *JournalEncoder) (ret *JournalEncoder, err error) {
		enc.cfg = cfg
		ret = enc
		return
	}
}

func WithFacility(facility syslog.Priority) Opt {
	return func(enc *JournalEncoder) (ret *JournalEncoder, err error) {
		enc.facility = facility
		ret = enc
		return
	}
}

func NewJournalEncoder(encoderConfig zapcore.EncoderConfig) (zapcore.Encoder, error) {
	return New(WithCfg(encoderConfig))
}

func init() {
	err := zap.RegisterEncoder("journal", NewJournalEncoder)
	if err != nil {
		panic("zap-journal: Failed to register encoder: " + err.Error())
	}
}
