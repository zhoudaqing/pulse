package mtproto

import (
	"crypto/rand"
	"encoding/binary"
	"log"
	"math"
	"math/big"
	"time"
)

func GenerateNonce(size int) []byte {
	b := make([]byte, size)
	_, _ = rand.Read(b)
	return b
}

func GenerateMessageId() int64 {
	const nano = 1000 * 1000 * 1000
	unixnano := time.Now().UnixNano()

	x := ((unixnano / nano) << 32) | ((unixnano % nano) & -3)
	mod := x & 3
	log.Println("mod of x", x, mod)
	if mod != 1 && mod != 3 {
		return GenerateMessageId()
	}

	return x
}

type EncodeBuf struct {
	buf []byte
}

func (e *EncodeBuf) Buf() []byte {
	return e.buf
}

func NewEncodeBuf(cap int) *EncodeBuf {
	return &EncodeBuf{make([]byte, 0, cap)}
}

func (e *EncodeBuf) Int(s int32) {
	e.buf = append(e.buf, 0, 0, 0, 0)
	binary.LittleEndian.PutUint32(e.buf[len(e.buf)-4:], uint32(s))
}

func (e *EncodeBuf) UInt(s uint32) {
	e.buf = append(e.buf, 0, 0, 0, 0)
	binary.LittleEndian.PutUint32(e.buf[len(e.buf)-4:], s)
}

func (e *EncodeBuf) Long(s int64) {
	e.buf = append(e.buf, 0, 0, 0, 0, 0, 0, 0, 0)
	binary.LittleEndian.PutUint64(e.buf[len(e.buf)-8:], uint64(s))
}

func (e *EncodeBuf) Double(s float64) {
	e.buf = append(e.buf, 0, 0, 0, 0, 0, 0, 0, 0)
	binary.LittleEndian.PutUint64(e.buf[len(e.buf)-8:], math.Float64bits(s))
}

func (e *EncodeBuf) String(s string) {
	e.StringBytes([]byte(s))
}

func (e *EncodeBuf) BigInt(s *big.Int) {
	e.StringBytes(s.Bytes())
}

func (e *EncodeBuf) StringBytes(s []byte) {
	var res []byte
	size := len(s)
	if size < 254 {
		nl := 1 + size + (4-(size+1)%4)&3
		res = make([]byte, nl)
		res[0] = byte(size)
		copy(res[1:], s)

	} else {
		nl := 4 + size + (4-size%4)&3
		res = make([]byte, nl)
		binary.LittleEndian.PutUint32(res, uint32(size<<8|254))
		copy(res[4:], s)

	}
	e.buf = append(e.buf, res...)
}

func (e *EncodeBuf) Bytes(s []byte) {
	e.buf = append(e.buf, s...)
}

func (e *EncodeBuf) VectorInt(v []int32) {
	x := make([]byte, 4+4+len(v)*4)
	binary.LittleEndian.PutUint32(x, crc_vector)
	binary.LittleEndian.PutUint32(x[4:], uint32(len(v)))
	i := 8
	for _, v := range v {
		binary.LittleEndian.PutUint32(x[i:], uint32(v))
		i += 4
	}
	e.buf = append(e.buf, x...)
}

func (e *EncodeBuf) VectorLong(v []int64) {
	x := make([]byte, 4+4+len(v)*8)
	binary.LittleEndian.PutUint32(x, crc_vector)
	binary.LittleEndian.PutUint32(x[4:], uint32(len(v)))
	i := 8
	for _, v := range v {
		binary.LittleEndian.PutUint64(x[i:], uint64(v))
		i += 8
	}
	e.buf = append(e.buf, x...)
}

func (e *EncodeBuf) VectorString(v []string) {
	x := make([]byte, 8)
	binary.LittleEndian.PutUint32(x, crc_vector)
	binary.LittleEndian.PutUint32(x[4:], uint32(len(v)))
	e.buf = append(e.buf, x...)
	for _, v := range v {
		e.String(v)
	}
}

func (e *EncodeBuf) Vector(v []TL) {
	x := make([]byte, 8)
	binary.LittleEndian.PutUint32(x, crc_vector)
	binary.LittleEndian.PutUint32(x[4:], uint32(len(v)))
	e.buf = append(e.buf, x...)
	for _, v := range v {
		e.buf = append(e.buf, v.encode()...)
	}
}

func (e TL_MT_message) encode() []byte {
	x := NewEncodeBuf(512)
	x.Long(e.Msg_id)
	x.Int(e.Seq_no)
	x.Int(e.Size)
	x.Bytes(e.Data.encode())

	return x.buf
}

// TODO: Does only server send messages below?
func (e TL_msg_container) encode() []byte {
	x := NewEncodeBuf(512)
	x.UInt(crc_msg_container)
	x.Int(int32(len(e.Items)))

	for _, m := range e.Items {
		// log.Println("data type:", reflect.TypeOf(m.Data))
		x.Bytes(m.encode())
	}

	return x.buf
}
func (e TL_resPQ) encode() []byte {
	x := NewEncodeBuf(512)
	x.UInt(crc_resPQ)
	x.Bytes(e.Nonce)
	x.Bytes(e.Server_nonce)
	x.BigInt(e.Pq)
	x.VectorLong(e.Fingerprints)

	return x.buf
}
func (e TL_server_DH_params_ok) encode() []byte {
	x := NewEncodeBuf(1024)
	x.UInt(crc_server_DH_params_ok)
	x.Bytes(e.Nonce)
	x.Bytes(e.Server_nonce)
	x.StringBytes(e.Encrypted_answer)

	return x.buf
}
func (e TL_server_DH_params_fail) encode() []byte { return nil }
func (e TL_server_DH_inner_data) encode() []byte {
	x := NewEncodeBuf(1024)
	x.UInt(crc_server_DH_inner_data)
	x.Bytes(e.Nonce)
	x.Bytes(e.Server_nonce)
	x.Int(e.G)
	x.BigInt(e.Dh_prime)
	x.BigInt(e.G_a)
	x.Int(e.Server_time)

	return x.buf
}
func (e TL_dh_gen_ok) encode() []byte {
	x := NewEncodeBuf(512)
	x.UInt(crc_dh_gen_ok)
	x.Bytes(e.Nonce)
	x.Bytes(e.Server_nonce)
	x.Bytes(e.New_nonce_hash1)
	return x.buf
}

func (e TL_http_wait) encode() []byte {
	x := NewEncodeBuf(512)
	x.UInt(crc_http_wait)
	x.Int(e.Max_delay)
	x.Int(e.Wait_after)
	x.Int(e.Max_wait)
	return x.buf
}

func (e TL_rpc_result) encode() []byte {
	x := NewEncodeBuf(1024)
	x.UInt(crc_rpc_result)
	x.Long(e.Req_msg_id)
	x.Bytes(EncodeInterface(e.Obj))

	return x.buf
}

func (e TL_rpc_error) encode() []byte { return nil }

func (e TL_new_session_created) encode() []byte {
	log.Println("encode TL_new_session_created")
	x := NewEncodeBuf(512)
	x.UInt(crc_new_session_created)
	x.Long(e.First_msg_id)
	x.Long(e.Unique_id)
	x.Bytes(e.Server_salt)

	return x.buf
}

func (e TL_bad_server_salt) encode() []byte      { return nil }
func (e TL_bad_msg_notification) encode() []byte { return nil }

func (e TL_req_pq) encode() []byte {
	x := NewEncodeBuf(20)
	x.UInt(crc_req_pq)
	x.Bytes(e.Nonce)
	return x.buf
}

func (e TL_p_q_inner_data) encode() []byte {
	x := NewEncodeBuf(256)
	x.UInt(crc_p_q_inner_data)
	x.BigInt(e.Pq)
	x.BigInt(e.P)
	x.BigInt(e.Q)
	x.Bytes(e.Nonce)
	x.Bytes(e.Server_nonce)
	x.Bytes(e.New_nonce)
	return x.buf
}

func (e TL_req_DH_params) encode() []byte {
	x := NewEncodeBuf(512)
	x.UInt(crc_req_DH_params)
	x.Bytes(e.Nonce)
	x.Bytes(e.Server_nonce)
	x.BigInt(e.P)
	x.BigInt(e.Q)
	x.Long(int64(e.Fp))
	x.StringBytes(e.Encdata)
	return x.buf
}

func (e TL_client_DH_inner_data) encode() []byte {
	x := NewEncodeBuf(512)
	x.UInt(crc_client_DH_inner_data)
	x.Bytes(e.Nonce)
	x.Bytes(e.Server_nonce)
	x.Long(e.Retry)
	x.BigInt(e.G_b)
	return x.buf
}

func (e TL_set_client_DH_params) encode() []byte {
	x := NewEncodeBuf(256)
	x.UInt(crc_set_client_DH_params)
	x.Bytes(e.Nonce)
	x.Bytes(e.Server_nonce)
	x.StringBytes(e.Encdata)
	return x.buf
}

func (e TL_ping) encode() []byte {
	x := NewEncodeBuf(32)
	x.UInt(crc_ping)
	x.Long(e.Ping_id)
	return x.buf
}

func (e TL_pong) encode() []byte {
	x := NewEncodeBuf(32)
	x.UInt(crc_pong)
	x.Long(e.Msg_id)
	x.Long(e.Ping_id)
	return x.buf
}

func (e TL_msgs_ack) encode() []byte {
	x := NewEncodeBuf(64)
	x.UInt(crc_msgs_ack)
	x.VectorLong(e.MsgIds)
	return x.buf
}

func (e TL_boolFalse) encode() []byte {
	x := NewEncodeBuf(4)
	x.UInt(crc_boolFalse)
	return x.buf
}

func (e TL_boolTrue) encode() []byte {
	x := NewEncodeBuf(4)
	x.UInt(crc_boolTrue)
	return x.buf
}

func (e TL_null) encode() []byte {
	x := NewEncodeBuf(4)
	x.UInt(crc_null)
	return x.buf
}
