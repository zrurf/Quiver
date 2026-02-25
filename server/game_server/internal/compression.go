package internal

import (
	"github.com/klauspost/compress/zstd"
)

type Compressor struct {
	encoder *zstd.Encoder
	decoder *zstd.Decoder
}

func NewCompressor(level int) *Compressor {
	enc, _ := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.EncoderLevelFromZstd(level)))
	dec, _ := zstd.NewReader(nil)
	return &Compressor{
		encoder: enc,
		decoder: dec,
	}
}

func (c *Compressor) Compress(data []byte) []byte {
	return c.encoder.EncodeAll(data, make([]byte, 0, len(data)))
}

func (c *Compressor) Decompress(data []byte) ([]byte, error) {
	return c.decoder.DecodeAll(data, nil)
}
