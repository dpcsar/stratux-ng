package gdl90

// crc16 implements the CRC used by GDL90 message integrity.
// This is a table-driven CRC-16 with polynomial 0x1021.
//
// NOTE: Different CRC-16 variants exist; this matches widely-used GDL90 framing
// behavior in the ecosystem.
func crc16(data []byte) uint16 {
	var crc uint16
	for _, b := range data {
		crc = crc16Table[crc>>8] ^ (crc << 8) ^ uint16(b)
	}
	return crc
}

var crc16Table = func() [256]uint16 {
	var table [256]uint16
	for i := 0; i < 256; i++ {
		crc := uint16(i) << 8
		for bit := 0; bit < 8; bit++ {
			if (crc & 0x8000) != 0 {
				crc = (crc << 1) ^ 0x1021
			} else {
				crc <<= 1
			}
		}
		table[i] = crc
	}
	return table
}()
