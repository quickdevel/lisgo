package lisgo

import "C"
import (
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"os"

	_ "image/gif"

	"github.com/apex/log"
	_ "github.com/sergeymakinen/go-bmp"
	_ "golang.org/x/image/tiff"
)

// PageReader represents a single page received from scanner
type PageReader struct {
	Width     int
	Height    int
	Format    uint32
	ImageSize uint
	Session   *ScanSession
}

// Read portion of data from scanner into a buffer
func (sb *PageReader) Read(p []byte) (int, error) {
	if sb.Session.EndOfPage() {
		return 0, io.EOF
	}

	return sb.Session.ScanRead(p)
}

// Read all page data
func (sb *PageReader) ReadToEnd() ([]byte, error) {
	var result []byte
	buf := make([]byte, 128*1024)

	for {
		n, err := sb.Read(buf)
		if n > 0 {
			result = append(result, buf[:n]...)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

func (sb *PageReader) readAndDecodeRawRGB24() (image.Image, error) {
	rawData, err := sb.ReadToEnd()
	if err != nil {
		return nil, err
	}

	width := sb.Width
	height := int(len(rawData) / 3 / sb.Width)

	// Create RGBA image with fixed width and height
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// left-top to right-bottom
			idx := (y*width + x) * 3

			// right-bottom to left-top
			// idx := ((height-1-y)*width + (width - 1 - x)) * 3

			img.Set(x, y, color.RGBA{
				R: rawData[idx],
				G: rawData[idx+1],
				B: rawData[idx+2],
				A: 255,
			})
		}
	}

	return img, nil
}

func (sb *PageReader) GetImage() (image.Image, error) {
	log.Debug("reading image data")

	var img image.Image
	var err error
	if sb.Format == LisImgFormatRawRGB24 {
		// Use custom function for RawRGB24
		img, err = sb.readAndDecodeRawRGB24()
	} else {
		// Try use std image package for other formats
		// supported formats: bmp, jpeg, gif, tiff
		img, _, err = image.Decode(sb)
	}

	log.Debug("image data is read")
	return img, err
}

func (sb *PageReader) WriteToFile(name string, format string) error {
	outputFile, err := os.Create(name)
	if err != nil {
		return err
	}
	defer outputFile.Close()

	img, err := sb.GetImage()
	if err != nil {
		return err
	}
	switch format {
	case "png":
		return png.Encode(outputFile, img)
	case "jpg":
		return jpeg.Encode(outputFile, img, &jpeg.Options{Quality: 50})
	}
	return errors.New("unknown file format")

}

// WriteToPng writes image to file
func (sb *PageReader) WriteToPng(name string) error {
	return sb.WriteToFile(name, "png")
}

// WriteToJpeg writes image to file
func (sb *PageReader) WriteToJpeg(name string) error {
	return sb.WriteToFile(name, "jpg")
}

// NewPageReader converts data buffer to image object
func NewPageReader(session *ScanSession, param *ScanParameters) *PageReader {

	b := PageReader{
		Width:     param.Width(),
		Height:    param.Height(),
		Format:    param.ImageFormat(),
		ImageSize: param.ImageSize(),
		Session:   session,
	}

	return &b

}
