// Package polyline implements a Google Maps Encoding Polyline encoder and decoder.
//
// See https://developers.google.com/maps/documentation/utilities/polylinealgorithm
package polyline

import (
	"errors"
	"math"
)

var (
	ErrDimensionalMismatch  = errors.New("dimensional mismatch")
	ErrInvalidByte          = errors.New("invalid byte")
	ErrUnterminatedSequence = errors.New("unterminated sequence")
)

func round(x float64) int {
	if x < 0 {
		return int(-math.Floor(-x + 0.5))
	} else {
		return int(math.Floor(x + 0.5))
	}
}

// A Codec represents an encoder.
type Codec struct {
	Dim   int     // Dimensionality, normally 2
	Scale float64 // Scale, normally 1e5
}

var defaultCodec = Codec{Dim: 2, Scale: 1e5}

// DecodeUint decodes a single unsigned integer from buf.
func DecodeUint(buf []byte) (uint, []byte, error) {
	var u, shift uint
	for i, b := range buf {
		switch {
		case 63 <= b && b < 95:
			u += (uint(b) - 63) << shift
			return u, buf[i+1:], nil
		case 95 <= b && b < 127:
			u += (uint(b) - 95) << shift
			shift += 5
		default:
			return 0, nil, ErrInvalidByte
		}
	}
	return 0, nil, ErrUnterminatedSequence
}

// DecodeInt decodes a single signed integer from buf.
func DecodeInt(buf []byte) (int, []byte, error) {
	u, buf, err := DecodeUint(buf)
	if err != nil {
		return 0, nil, err
	}
	if u&1 == 0 {
		return int(u >> 1), buf, nil
	} else {
		return -int((u + 1) >> 1), buf, nil
	}
}

// EncodeUint appends the encoding of a single unsigned integer u to buf.
func EncodeUint(buf []byte, u uint) []byte {
	for u >= 32 {
		buf = append(buf, byte((u&31)+95))
		u = u >> 5
	}
	buf = append(buf, byte(u+63))
	return buf
}

// EncodeInt appends the encoding of a single signed integer i to buf.
func EncodeInt(buf []byte, i int) []byte {
	var u uint
	if i < 0 {
		u = uint(^(i << 1))
	} else {
		u = uint(i << 1)
	}
	return EncodeUint(buf, u)
}

// DecodeCoord decodes a single coordinate from buf.
func (c Codec) DecodeCoord(buf []byte) ([]float64, []byte, error) {
	coord := make([]float64, c.Dim)
	for i := range coord {
		var err error
		var j int
		j, buf, err = DecodeInt(buf)
		if err != nil {
			return nil, nil, err
		}
		coord[i] = float64(j) / c.Scale
	}
	return coord, buf, nil
}

// DecodeCoords decodes an array of coordinates from buf.
func (c Codec) DecodeCoords(buf []byte) ([][]float64, []byte, error) {
	var coord []float64
	var err error
	coord, buf, err = c.DecodeCoord(buf)
	if err != nil {
		return nil, nil, err
	}
	coords := [][]float64{coord}
	for i := 1; len(buf) > 0; i++ {
		coord, buf, err = c.DecodeCoord(buf)
		if err != nil {
			return nil, nil, err
		}
		for j := range coord {
			coord[j] += coords[i-1][j]
		}
		coords = append(coords, coord)
	}
	return coords, nil, nil
}

// DecodeFlatCoords decodes coordinates from buf, appending them to a
// one-dimensional array.
func (c Codec) DecodeFlatCoords(fcs []float64, buf []byte) ([]float64, []byte, error) {
	if len(fcs)%c.Dim != 0 {
		return nil, nil, ErrDimensionalMismatch
	}
	last := make([]int, c.Dim)
	for len(buf) > 0 {
		for j := 0; j < c.Dim; j++ {
			var err error
			var k int
			k, buf, err = DecodeInt(buf)
			if err != nil {
				return nil, nil, err
			}
			last[j] += k
			fcs = append(fcs, float64(last[j])/c.Scale)
		}
	}
	if len(fcs)%c.Dim != 0 {
		return nil, nil, ErrDimensionalMismatch
	}
	return fcs, nil, nil
}

// EncodeCoord encodes a single coordinate to buf.
func (c Codec) EncodeCoord(buf []byte, coord []float64) []byte {
	for _, x := range coord {
		buf = EncodeInt(buf, round(c.Scale*x))
	}
	return buf
}

// EncodeCoords appends the encoding of an array of coordinates coords to buf.
func (c Codec) EncodeCoords(buf []byte, coords [][]float64) []byte {
	last := make([]int, c.Dim)
	for _, coord := range coords {
		for i, x := range coord {
			ex := round(c.Scale * x)
			buf = EncodeInt(buf, ex-last[i])
			last[i] = ex
		}
	}
	return buf
}

// EncodeFlatCoords encodes a one-dimensional array of coordinates to buf.
func (c Codec) EncodeFlatCoords(buf []byte, fcs []float64) ([]byte, error) {
	if len(fcs)%c.Dim != 0 {
		return nil, ErrDimensionalMismatch
	}
	last := make([]int, c.Dim)
	for i, x := range fcs {
		ex := round(c.Scale * x)
		buf = EncodeInt(buf, ex-last[i%c.Dim])
		last[i%c.Dim] = ex
	}
	return buf, nil
}

// DecodeCoord decodes a single coordinate from buf.
func DecodeCoord(buf []byte) ([]float64, []byte, error) {
	return defaultCodec.DecodeCoord(buf)
}

// DecodeCoords decodes an array of coordinates from buf.
func DecodeCoords(buf []byte) ([][]float64, []byte, error) {
	return defaultCodec.DecodeCoords(buf)
}

// EncodeCoord returns the encoding of an array of coordinates.
func EncodeCoord(coord []float64) []byte {
	return defaultCodec.EncodeCoord(nil, coord)
}

// EncodeCoords returns the encoding of an array of coordinates.
func EncodeCoords(coords [][]float64) []byte {
	return defaultCodec.EncodeCoords(nil, coords)
}
