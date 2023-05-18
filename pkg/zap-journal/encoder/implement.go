package encoder

import (
	"encoding/hex"
	"log/syslog"
	"strings"
	"time"

	"github.com/black-desk/deepin-network-proxy-manager/pkg/zap-journal/internal/bufferpool"
	"github.com/black-desk/deepin-network-proxy-manager/pkg/zap-journal/internal/zapjson"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

var _ zapcore.Encoder = &JournalEncoder{}

func (enc *JournalEncoder) beforeJsonEncoder(key string) {
	enc.addKey(key)
	enc.leaveLen()
}

func (enc *JournalEncoder) addKey(key string) {
	for i := range enc.nss {
		enc.buf.AppendString(strings.ToUpper(enc.nss[i]))
		enc.buf.AppendString("_")
	}
	enc.buf.AppendString(strings.ToUpper(key))
}

func (enc *JournalEncoder) addEqual() {
	enc.buf.AppendByte(byte('='))
}

func (enc *JournalEncoder) addNewline() {
	enc.buf.AppendString("\n")
}

func (enc *JournalEncoder) leaveLen() {
	enc.buf.AppendString("\n")
	enc.lengthPos = enc.buf.Len()
	enc.buf.Write([]byte{0, 0, 0, 0, 0, 0, 0, 0})
}

func (enc *JournalEncoder) afterJsonEncoder() {
	enc.fillLen()
	enc.addNewline()
}

func (enc *JournalEncoder) fillLen() {
	length := enc.buf.Len() - enc.lengthPos - 8

	bs := enc.buf.Bytes()
	bs[enc.lengthPos] = byte(length)
	bs[enc.lengthPos+1] = byte(length >> 8)
	bs[enc.lengthPos+2] = byte(length >> 16)
	bs[enc.lengthPos+3] = byte(length >> 24)
	bs[enc.lengthPos+4] = byte(length >> 32)
	bs[enc.lengthPos+5] = byte(length >> 40)
	bs[enc.lengthPos+6] = byte(length >> 48)
	bs[enc.lengthPos+7] = byte(length >> 56)
}

func (enc *JournalEncoder) AddArray(key string, marshaler zapcore.ArrayMarshaler) (err error) {
	enc.beforeJsonEncoder(key)
	err = enc.jenc.AppendArray(marshaler)
	enc.afterJsonEncoder()
	return
}

func (enc *JournalEncoder) AddBinary(key string, value []byte) {
	enc.addKey(key)
	enc.leaveLen()

	length := len(value)
	length = hex.EncodedLen(length)
	pos := enc.buf.Len()
	enc.buf.Write(make([]byte, length))
	dist := enc.buf.Bytes()[pos:]
	hex.Encode(dist, value)

	enc.fillLen()
}

func (enc *JournalEncoder) AddBool(key string, value bool) {
	enc.addKey(key)
	enc.addEqual()
	enc.jenc.AppendBool(value)
	enc.addNewline()
}

func (enc *JournalEncoder) AddByteString(key string, value []byte) {
	enc.beforeJsonEncoder(key)
	enc.jenc.AppendByteString(value)
	enc.afterJsonEncoder()
}

func (enc *JournalEncoder) AddComplex128(key string, value complex128) {
	enc.addKey(key)
	enc.addEqual()
	enc.jenc.AppendComplex128(value)
	enc.addNewline()
}

func (enc *JournalEncoder) AddComplex64(key string, value complex64) {
	enc.addKey(key)
	enc.addEqual()
	enc.jenc.AppendComplex64(value)
	enc.addNewline()
}

func (enc *JournalEncoder) AddDuration(key string, value time.Duration) {
	enc.addKey(key)
	enc.addEqual()
	enc.jenc.AppendDuration(value)
	enc.addNewline()
}

func (enc *JournalEncoder) AddFloat32(key string, value float32) {
	enc.addKey(key)
	enc.addEqual()
	enc.jenc.AppendFloat32(value)
	enc.addNewline()
}

func (enc *JournalEncoder) AddFloat64(key string, value float64) {
	enc.addKey(key)
	enc.addEqual()
	enc.jenc.AppendFloat64(value)
	enc.addNewline()
}

func (enc *JournalEncoder) AddInt(key string, value int) {
	enc.addKey(key)
	enc.addEqual()
	enc.jenc.AppendInt(value)
	enc.addNewline()
}

func (enc *JournalEncoder) AddInt16(key string, value int16) {
	enc.addKey(key)
	enc.addEqual()
	enc.jenc.AppendInt16(value)
	enc.addNewline()
}

func (enc *JournalEncoder) AddInt32(key string, value int32) {
	enc.addKey(key)
	enc.addEqual()
	enc.jenc.AppendInt32(value)
	enc.addNewline()
}

func (enc *JournalEncoder) AddInt64(key string, value int64) {
	enc.addKey(key)
	enc.addEqual()
	enc.jenc.AppendInt64(value)
	enc.addNewline()
}

func (enc *JournalEncoder) AddInt8(key string, value int8) {
	enc.addKey(key)
	enc.addEqual()
	enc.jenc.AppendInt8(value)
	enc.addNewline()
}

func (enc *JournalEncoder) AddObject(key string, marshaler zapcore.ObjectMarshaler) (err error) {
	enc.beforeJsonEncoder(key)
	err = enc.jenc.AppendObject(marshaler)
	enc.afterJsonEncoder()
	return
}

func (enc *JournalEncoder) AddReflected(key string, value interface{}) (err error) {
	enc.beforeJsonEncoder(key)
	err = enc.jenc.AppendReflected(value)
	enc.afterJsonEncoder()
	return
}

func (enc *JournalEncoder) AddString(key string, value string) {
	enc.beforeJsonEncoder(key)
	enc.buf.AppendString(value)
	enc.afterJsonEncoder()
}

func (enc *JournalEncoder) AddTime(key string, value time.Time) {
	enc.beforeJsonEncoder(key)
	enc.jenc.AppendTime(value)
	enc.afterJsonEncoder()
}

func (enc *JournalEncoder) AddUint(key string, value uint) {
	enc.addKey(key)
	enc.addEqual()
	enc.jenc.AppendUint(value)
	enc.addNewline()
}

func (enc *JournalEncoder) AddUint16(key string, value uint16) {
	enc.addKey(key)
	enc.addEqual()
	enc.jenc.AppendUint16(value)
	enc.addNewline()
}

func (enc *JournalEncoder) AddUint32(key string, value uint32) {
	enc.addKey(key)
	enc.addEqual()
	enc.jenc.AppendUint32(value)
	enc.addNewline()
}

func (enc *JournalEncoder) AddUint64(key string, value uint64) {
	enc.addKey(key)
	enc.addEqual()
	enc.jenc.AppendUint64(value)
	enc.addNewline()
}

func (enc *JournalEncoder) AddUint8(key string, value uint8) {
	enc.addKey(key)
	enc.addEqual()
	enc.jenc.AppendUint8(value)
	enc.addNewline()
}

func (enc *JournalEncoder) AddUintptr(key string, value uintptr) {
	enc.addKey(key)
	enc.addEqual()
	enc.jenc.AppendUintptr(value)
	enc.addNewline()
}

func (enc *JournalEncoder) OpenNamespace(key string) {
	enc.nss = append(enc.nss, key)
}

func (enc *JournalEncoder) Clone() (ret zapcore.Encoder) {
	return enc.clone()
}

func (enc *JournalEncoder) clone() (ret *JournalEncoder) {
	nenc := journalEncoderPool.Get().(*JournalEncoder)

	*nenc = JournalEncoder{
		facility:  enc.facility,
		cfg:       enc.cfg,
		jenc:      nil,
		buf:       bufferpool.Get(),
		lengthPos: 0,
		nss:       enc.nss,
	}

	nenc.buf.Write(enc.buf.Bytes())
	nenc.jenc = zapjson.New(
		nenc.cfg,
		nenc.buf,
	)

	ret = nenc
	return
}

func (enc *JournalEncoder) EncodeEntry(
	entry zapcore.Entry, fields []zapcore.Field,
) (
	buf *buffer.Buffer, err error,
) {
	nenc := enc.clone()

	nenc.buf.AppendString("PRIORITY=")
	switch entry.Level {
	case zapcore.DebugLevel:
		nenc.buf.AppendUint(uint64(syslog.LOG_DEBUG))
	case zapcore.InfoLevel:
		nenc.buf.AppendUint(uint64(syslog.LOG_INFO))
	case zapcore.WarnLevel:
		nenc.buf.AppendUint(uint64(syslog.LOG_WARNING))
	case zapcore.ErrorLevel:
		nenc.buf.AppendUint(uint64(syslog.LOG_ERR))
	case zapcore.DPanicLevel:
		nenc.buf.AppendUint(uint64(syslog.LOG_CRIT))
	case zapcore.PanicLevel:
		nenc.buf.AppendUint(uint64(syslog.LOG_CRIT))
	case zapcore.FatalLevel:
		nenc.buf.AppendUint(uint64(syslog.LOG_EMERG))
	default:
		panic("this should never happened")
	}
	nenc.buf.AppendString("\n")

	nenc.buf.AppendString("SYSLOG_FACILITY=")
	nenc.buf.AppendUint(uint64(nenc.facility))
	nenc.buf.AppendString("\n")

	if !entry.Time.IsZero() {
		nenc.buf.AppendString("SYSLOG_TIMESTAMP=")
		nenc.buf.AppendString(entry.Time.Format(time.Stamp))
		nenc.buf.AppendString("\n")
	}

	nenc.beforeJsonEncoder("SYSLOG_IDENTIFIER")
	if entry.LoggerName != "" {
		nenc.buf.AppendString(entry.LoggerName)
	} else {
		nenc.buf.AppendString("")
	}
	nenc.afterJsonEncoder()

	if entry.Caller.Defined {
		nenc.buf.AppendString("CODE_FILE=")
		nenc.buf.AppendString(entry.Caller.File)
		nenc.buf.AppendString("\n")
		nenc.buf.AppendString("CODE_LINE=")
		nenc.buf.AppendUint(uint64(entry.Caller.Line))
		nenc.buf.AppendString("\n")
		nenc.buf.AppendString("CODE_FUNC=")
		nenc.buf.AppendString(entry.Caller.Function)
		nenc.buf.AppendString("\n")
	}

	if entry.Message != "" {
		nenc.beforeJsonEncoder("MESSAGE")
		nenc.buf.AppendString(entry.Message)
		nenc.afterJsonEncoder()
	}

	if entry.Stack != "" && nenc.cfg.StacktraceKey != "" {
		nenc.beforeJsonEncoder(nenc.cfg.StacktraceKey)
		nenc.buf.AppendString(entry.Stack)
		nenc.afterJsonEncoder()
	}

	for i := range fields {
		fields[i].AddTo(nenc)
	}

	buf = nenc.buf
	nenc.buf = nil

	journalEncoderPool.Put(nenc)

	return
}
