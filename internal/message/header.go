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
	OPcodeBit = 11 // 	Opcode (4 bits)
	AABit     = 10 //  (Authoritative Answer)
	TCBit     = 9  //  (Truncated)
	RDBit     = 8  //  (Recursion Desired)
	RABit     = 7  //  (Recursion Available)
	ZBit      = 4  // 	(Reserved, 3 bits)
	RcodeBit  = 0  // 	(Response Code, 4 bits)
)

type Header struct {
	ID      uint16
	Flags   uint16
	Qdcount uint16
	Ancount uint16
	Nscount uint16
	Arcount uint16
}

func HandleHeader(data []byte) (Header, error) {
	if len(data) < 12 {
		return Header{}, errors.New("DNS message is too short")
	}

	h := Header{
		ID:      binary.BigEndian.Uint16(data[0:2]),
		Flags:   binary.BigEndian.Uint16(data[2:4]),
		Qdcount: binary.BigEndian.Uint16(data[4:6]),
		Ancount: binary.BigEndian.Uint16(data[6:8]),
		Nscount: binary.BigEndian.Uint16(data[8:10]),
		Arcount: binary.BigEndian.Uint16(data[10:12]),
	}

	QR, OPcode, AA, TC, RD, RA, Z, Rcode := h.parseFlags(&h.Flags)

	if h.Qdcount == 0 {
		return Header{}, errors.New("QDCOUNT is 0")
	}

	if OPcode != 0 {
		Rcode = 4
	}

	log.Println("QR:", QR, " OPCODE:", OPcode, " AA:", AA, " TC:", TC, " RD:", RD, " RA:", RA, " Z:", Z, " RCODE:", Rcode)

	switch Rcode {
	case 1:
		return Header{}, ErrFormatError
	case 2:
		return Header{}, ErrServerFailure
	case 3:
		return Header{}, ErrNameError
	case 4:
		return Header{}, ErrNotImplemented
	case 5:
		return Header{}, ErrRefused
	case 6, 7, 8, 9, 10, 11, 12, 13, 14, 15:
		return Header{}, ErrUnSupported
	}

	return h, nil
}

func (h Header) parseFlags(flags *uint16) (QR, OPcode, AA, TC, RD, RA, Z, Rcode uint16) {
	// QR (15)
	QR = (*flags >> 15) & 0x1
	// OPcode (11-14)
	OPcode = (*flags >> 11) & 0xF
	// AA (10)
	AA = (*flags >> 10) & 0x1
	// TC (9)
	TC = (*flags >> 9) & 0x1
	// RD (8)
	RD = (*flags >> 8) & 0x1
	// RA (7)
	RA = (*flags >> 7) & 0x1
	// Z (4-6)
	Z = (*flags >> 4) & 0x7
	// Rcode (0-3)
	Rcode = *flags & 0xF
	return
}

func (h Header) SetFlags(flags *uint16, QR, OPcode, AA, TC, RD, RA, Z, Rcode uint16) {
	*flags = 0

	*flags |= QR << QRBit
	*flags |= (OPcode & 0xF) << OPcodeBit // Opcode - 4 bits
	*flags |= AA << AABit
	*flags |= TC << TCBit
	*flags |= RD << RDBit
	*flags |= RA << RABit
	*flags |= (Z & 0x7) << ZBit // Z - 3 bits
	*flags |= Rcode & 0xF       // Rcode - 4 bits
}

func (h Header) Decode() []byte {
	data := bytes.NewBuffer(nil)

	binary.Write(data, binary.BigEndian, h.ID)
	binary.Write(data, binary.BigEndian, h.Flags)
	binary.Write(data, binary.BigEndian, h.Qdcount)
	binary.Write(data, binary.BigEndian, h.Ancount)
	binary.Write(data, binary.BigEndian, h.Nscount)
	binary.Write(data, binary.BigEndian, h.Arcount)

	return data.Bytes()
}
