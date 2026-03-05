package message

import (
	"bytes"
	"encoding/binary"
	"errors"
	"log"
)

var (
	ErrFormatError    = errors.New("the server was unable to interpret the query")
	ErrServerFailure  = errors.New("the name server was unable to process this query due to a problem with the name server")
	ErrNameError      = errors.New("the domain name referenced in the query does not exist")
	ErrNotImplemented = errors.New("the name server doesn't support the requested kind of query")
	ErrRefused        = errors.New("the name server refuses to perform the specified operation for policy reasons")
	ErrUnSupported    = errors.New("the unsupported option opcode (reserved for future)")
)

const (
	QRBit     = 15 //  (Query/Response)
	OPcodeBit = 11 //  (Opcode, 4 bits)
	AABit     = 10 //  (Authoritative Answer)
	TCBit     = 9  //  (Truncated)
	RDBit     = 8  //  (Recursion Desired)
	RABit     = 7  //  (Recursion Available)
	ZBit      = 4  //  (Reserved, 3 bits)
	RcodeBit  = 0  //  (Response Code, 4 bits)
)

type Header struct {
	ID      uint16 // ID of record
	Flags   uint16 //
	Qdcount uint16 // number of questions
	Ancount uint16 // number of answers
	Nscount uint16 // number of authority records
	Arcount uint16 // number of additional records
}

const (
	ErrShortMsg    = "DNS message is too short"
	ErrQdcountZero = "QDCOUNT is 0"
)

func HandleHeader(data []byte) (*Header, error) {
	if len(data) < 12 {
		return &Header{}, errors.New(ErrShortMsg)
	}

	h := &Header{
		ID:      binary.BigEndian.Uint16(data[0:2]),
		Flags:   binary.BigEndian.Uint16(data[2:4]),
		Qdcount: binary.BigEndian.Uint16(data[4:6]),
		Ancount: binary.BigEndian.Uint16(data[6:8]),
		Nscount: binary.BigEndian.Uint16(data[8:10]),
		Arcount: binary.BigEndian.Uint16(data[10:12]),
	}

	QR, OPcode, AA, TC, RD, RA, Z, Rcode := h.parseFlags()

	if h.Qdcount == 0 {
		return &Header{}, errors.New(ErrQdcountZero)
	}

	if OPcode != 0 {
		Rcode = 4
	}

	log.Println("QR:", QR, " OPCODE:", OPcode, " AA:", AA, " TC:", TC, " RD:", RD, " RA:", RA, " Z:", Z, " RCODE:", Rcode)

	switch Rcode {
	case 1:
		return nil, ErrFormatError
	case 2:
		return nil, ErrServerFailure
	case 3:
		return nil, ErrNameError
	case 4:
		return nil, ErrNotImplemented
	case 5:
		return nil, ErrRefused
	case 6, 7, 8, 9, 10, 11, 12, 13, 14, 15:
		return nil, ErrUnSupported
	}

	return h, nil
}

func (h *Header) parseFlags() (QR, OPcode, AA, TC, RD, RA, Z, Rcode uint8) {
	QR = uint8((h.Flags >> 15) & 1)
	OPcode = uint8((h.Flags >> 11) & 0xF)
	AA = uint8((h.Flags >> 10) & 0x1)
	TC = uint8((h.Flags >> 9) & 0x1)
	RD = uint8((h.Flags >> 8) & 0x1)
	RA = uint8((h.Flags >> 7) & 0x1)
	Z = uint8((h.Flags >> 4) & 0x7)
	Rcode = uint8(h.Flags & 0xF)
	return
}

func (h *Header) SetFlags(QR, OPcode, AA, TC, RD, RA, Z, Rcode uint8) uint16 {
	h.Flags = 0

	h.Flags |= uint16(QR&1) << QRBit
	h.Flags |= uint16(OPcode&0xF) << OPcodeBit
	h.Flags |= uint16(AA&1) << AABit
	h.Flags |= uint16(TC&1) << TCBit
	h.Flags |= uint16(RD&1) << RDBit
	h.Flags |= uint16(RA&1) << RABit
	h.Flags |= uint16(Z&0x7) << ZBit
	h.Flags |= uint16(Rcode & 0xF)

	return h.Flags
}

func (h *Header) Decode() ([]byte, error) {
	data := bytes.NewBuffer(nil)

	binary.Write(data, binary.BigEndian, h.ID)
	binary.Write(data, binary.BigEndian, h.Flags)
	binary.Write(data, binary.BigEndian, h.Qdcount)
	binary.Write(data, binary.BigEndian, h.Ancount)
	binary.Write(data, binary.BigEndian, h.Nscount)
	binary.Write(data, binary.BigEndian, h.Arcount)

	return data.Bytes(), nil
}
