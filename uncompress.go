package main

import (
	"bytes"
	"fmt"
)

const maxUncompressedResourceSize = 64 << 20

var (
	nextBit     int
	current     uint8
	inOffset    uint32
	maxInOffset uint32
)

type TCodeTableEntry struct {
	prefix uint16
	append uint8
}

func getByte(buf *bytes.Reader) uint8 {
	if inOffset >= maxInOffset {
		return 0
	} else {
		inOffset++
		b, err := buf.ReadByte()
		if err != nil {
			// Mimic (uint8)fgetc(f) when fgetc returns EOF (-1)
			return 0xFF
		}
		return b
	}
}

func getBits(buf *bytes.Reader, n uint32) uint16 {
	if n == 0 {
		return 0
	}

	x := uint32(0)
	for i := uint32(0); i < n; i++ {
		if (current & (1 << nextBit)) != 0 {
			x |= uint32(1) << i
		}
		nextBit++

		if nextBit > 7 {
			current = getByte(buf)
			nextBit = 0
		}
	}

	return uint16(x)
}

func uncompressLZW(buf *bytes.Reader, inSize, outSize uint32) []byte {
	validateCompressedSizes(buf, inSize, outSize, "LZW")
	outData := make([]byte, outSize)

	stackPtr := uint32(0)
	nBits := uint8(9)
	freeEntry := uint32(257)

	var (
		decodeStack [4096]uint8
		codeTable   [4096]TCodeTableEntry
		oldCode     uint16
		lastByte    uint16
		bitPos      uint32
		outOffset   uint32
	)

	maxInOffset = inSize
	nextBit = 0
	inOffset = 0
	current = getByte(buf)
	tmpBits := getBits(buf, uint32(nBits))
	lastByte = tmpBits
	oldCode = tmpBits

	outData[outOffset] = uint8(oldCode)
	outOffset++

	for inOffset < inSize {
		newCode := getBits(buf, uint32(nBits))
		bitPos += uint32(nBits)

		if newCode == 256 {
			nBits3 := uint32(nBits << 3)
			nSkip := (nBits3 - ((bitPos - 1) % nBits3)) - 1
			getBits(buf, nSkip)
			nBits = 9
			freeEntry = 256
			bitPos = 0
		} else {
			code := newCode

			if uint32(code) > freeEntry {
				panicDecompression("LZW", "code %d exceeds next dictionary entry %d", code, freeEntry)
			}
			if uint32(code) >= freeEntry {
				if stackPtr >= uint32(len(decodeStack)) {
					panicDecompression("LZW", "decode stack overflow")
				}

				decodeStack[stackPtr] = uint8(lastByte)
				stackPtr++
				code = oldCode
			}

			for code > 255 {
				if code > 4095 {
					panicDecompression("LZW", "dictionary code %d is out of range", code)
				}
				if stackPtr >= uint32(len(decodeStack)) {
					panicDecompression("LZW", "decode stack overflow")
				}
				decodeStack[stackPtr] = codeTable[code].append
				stackPtr++
				code = codeTable[code].prefix
			}

			if stackPtr >= uint32(len(decodeStack)) {
				panicDecompression("LZW", "decode stack overflow")
			}
			decodeStack[stackPtr] = uint8(code)
			stackPtr++
			lastByte = code

			for stackPtr > 0 {

				stackPtr--

				if outOffset >= outSize {
					return outData
				}

				outData[outOffset] = decodeStack[stackPtr]
				outOffset++
			}

			if freeEntry < 4096 {
				codeTable[freeEntry].prefix = oldCode
				codeTable[freeEntry].append = uint8(lastByte)
				freeEntry++
				temp := uint32(1 << nBits)

				if freeEntry >= temp && nBits < 12 {
					nBits++
					bitPos = 0
				}
			}

			oldCode = newCode
		}
	}

	if inOffset != inSize {
		panicDecompression("LZW", "consumed %d of %d input bytes", inOffset, inSize)
	}
	return outData
}

func uncompressRLE(buf *bytes.Reader, inSize, outSize uint32) []byte {
	validateCompressedSizes(buf, inSize, outSize, "RLE")
	outData := make([]byte, outSize)
	var outOffset uint32

	inOffset = 0
	maxInOffset = inSize
	readInput := func() uint8 {
		if inOffset >= inSize {
			panicDecompression("RLE", "input ended at byte %d", inOffset)
		}
		value, err := buf.ReadByte()
		if err != nil {
			panicDecompression("RLE", "read byte %d: %v", inOffset, err)
		}
		inOffset++
		return value
	}

	for outOffset < outSize {
		control := readInput()

		if (control & 0x80) == 0x80 {
			length := uint32(control & 0x7F)
			b := readInput()
			if outOffset+length > outSize {
				panicDecompression("RLE", "run of %d bytes exceeds output size %d at offset %d", length, outSize, outOffset)
			}

			for i := uint32(0); i < length; i++ {
				outData[outOffset] = b
				outOffset++
			}
		} else {
			length := uint32(control)
			if outOffset+length > outSize {
				panicDecompression("RLE", "literal of %d bytes exceeds output size %d at offset %d", length, outSize, outOffset)
			}
			if inOffset+length > inSize {
				panicDecompression("RLE", "literal of %d bytes exceeds input size %d at offset %d", length, inSize, inOffset)
			}
			for i := uint32(0); i < length; i++ {
				outData[outOffset] = readInput()
				outOffset++
			}
		}
	}

	if inOffset != inSize {
		panicDecompression("RLE", "consumed %d of %d input bytes", inOffset, inSize)
	}

	return outData
}

func validateCompressedSizes(buf *bytes.Reader, inSize, outSize uint32, method string) {
	if outSize == 0 {
		panicDecompression(method, "declared output size is zero")
	}
	if outSize > maxUncompressedResourceSize {
		panicDecompression(method, "declared output size %d exceeds safety limit %d", outSize, maxUncompressedResourceSize)
	}
	if uint64(inSize) > uint64(buf.Len()) {
		panicDecompression(method, "declared input size %d exceeds remaining data %d", inSize, buf.Len())
	}
}

func panicDecompression(method, format string, args ...any) {
	panic(fmt.Sprintf("invalid %s compressed resource: %s", method, fmt.Sprintf(format, args...)))
}

func uncompress(buf *bytes.Reader, compressionMethod uint8, inSize, outSize uint32) []byte {
	switch compressionMethod {
	case 1:
		return uncompressRLE(buf, inSize, outSize)
	case 2:
		return uncompressLZW(buf, inSize, outSize)
	default:
		panic("unknown compression method")
	}
}
