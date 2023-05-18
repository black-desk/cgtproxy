package encoder_test

import (
	"fmt"
	"log/syslog"
	"testing"

	. "github.com/black-desk/deepin-network-proxy-manager/pkg/ginkgo-helper"
	"github.com/black-desk/deepin-network-proxy-manager/pkg/zap-journal/encoder"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

var _ = Describe("Systemd journal encoder for zap", func() {
	ContextTable("create a JournalEncoder", func(constructor func() zapcore.Encoder) {
		var (
			enc zapcore.Encoder
			err error
		)
		BeforeEach(func() {
			enc = constructor()
		})
		ContextTable("encode %s", func(field zapcore.Field, result []byte) {
			var buf *buffer.Buffer
			BeforeEach(func() {
				buf, err = enc.EncodeEntry(zapcore.Entry{}, []zapcore.Field{field})
				Expect(err).To(Succeed())
			})
			It("should produce a correct buffer", func() {
				Expect(string(buf.Bytes())).To(Equal(string(result)))
			})
		},
			ContextTableEntry(zap.Int32("SOME_INT", 1), []byte(fmt.Sprintf(""+
				"PRIORITY=%d\n"+
				"SYSLOG_FACILITY=%d\n"+
				"SYSLOG_IDENTIFIER\n"+
				"\000\000\000\000\000\000\000\000"+
				"\n"+
				"SOME_INT=%d\n",
				syslog.LOG_INFO, syslog.LOG_USER, 1,
			))).
				WithFmt("a int"),
			ContextTableEntry(zap.String("SOME_STRING", "a"), []byte(fmt.Sprintf(""+
				"PRIORITY=%d\n"+
				"SYSLOG_FACILITY=%d\n"+
				"SYSLOG_IDENTIFIER\n"+
				"\000\000\000\000\000\000\000\000"+
				"\n"+
				"SOME_STRING\n"+
				"\003\000\000\000\000\000\000\000"+
				"\"%s\"\n",
				syslog.LOG_INFO, syslog.LOG_USER, "a",
			))).
				WithFmt("a string"),
			ContextTableEntry(
				zap.Reflect("SOME_STRUCT",
					struct {
						Field1 int
						Field2 string
					}{
						Field1: 1,
						Field2: "string content",
					}),
				[]byte(fmt.Sprintf(""+
					"PRIORITY=%d\n"+
					"SYSLOG_FACILITY=%d\n"+
					"SYSLOG_IDENTIFIER\n"+
					"\000\000\000\000\000\000\000\000"+
					"\n"+
					"SOME_STRUCT\n"+
					"\046\000\000\000\000\000\000\000"+
					"%s\n",
					syslog.LOG_INFO, syslog.LOG_USER,
					`{"Field1":1,"Field2":"string content"}`,
				)),
			).WithFmt("a struct"),
			ContextTableEntry(
				zap.Reflect("SOME_MAP",
					map[string]string{
						"key1": "value1",
						"key2": "value2",
						"key3": "value3",
					}),
				[]byte(fmt.Sprintf(""+
					"PRIORITY=%d\n"+
					"SYSLOG_FACILITY=%d\n"+
					"SYSLOG_IDENTIFIER\n"+
					"\000\000\000\000\000\000\000\000"+
					"\n"+
					"SOME_MAP\n\061\000\000\000\000\000\000\000%s\n",
					syslog.LOG_INFO, syslog.LOG_USER,
					`{"key1":"value1","key2":"value2","key3":"value3"}`,
				)),
			).WithFmt("a map"),
			ContextTableEntry(
				zap.Reflect("SOME_ARRAY",
					[]string{"string1", "string2", "string3"}),
				[]byte(fmt.Sprintf(""+
					"PRIORITY=%d\n"+
					"SYSLOG_FACILITY=%d\n"+
					"SYSLOG_IDENTIFIER\n"+
					"\000\000\000\000\000\000\000\000"+
					"\n"+
					"SOME_ARRAY\n"+
					"\037\000\000\000\000\000\000\000"+
					"%s\n",
					syslog.LOG_INFO, syslog.LOG_USER,
					`["string1","string2","string3"]`,
				)),
			).WithFmt("a array"),
		)
	},
		ContextTableEntry(func() zapcore.Encoder {
			enc, err := encoder.New(
				encoder.WithCfg(zap.NewProductionEncoderConfig()),
			)
			Expect(err).To(Succeed())
			return enc
		}).WithFmt("zap-journal/encoder.New"),
		ContextTableEntry(func() zapcore.Encoder {
			enc, err := encoder.NewJournalEncoder(zap.NewProductionEncoderConfig())
			Expect(err).To(Succeed())
			return enc
		}).WithFmt("zap-journal/encoder.NewJournalEncoder"),
	)
})

func TestJournal(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Systemd-journal plugin for zap Suite")
}
