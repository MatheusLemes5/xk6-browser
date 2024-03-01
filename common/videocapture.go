package common

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"time"
)

// VideoCapturePersister defines the interface for persisting a video capture
type VideoCapturePersister interface {
	Persist(ctx context.Context, path string, data io.Reader) (err error)
}

type VideoFrame struct{
	Content   io.Reader
	Timestamp time.Time
}

// VideoFormat represents a video file format.
type VideoFormat string

// Valid video format options.
const (
	// VideoFormatPEG stores video as a series of jpeg files
	VideoFormatJPEG VideoFormat = "jpeg"
)

// String returns the video format as a string
func (f VideoFormat) String() string {
	return f.String()
}

var videoFormatToID = map[string]VideoFormat{ //nolint:gochecknoglobals
	"jpeg": VideoFormatJPEG,
}

type videocapture struct {
	ctx       context.Context
	persister VideoCapturePersister
	path      string
}

// creates a new videocapture for a session
func newVideoCapture(
	ctx context.Context,
	path string,
	persister VideoCapturePersister,
) *videocapture {
	return &videocapture{
		ctx:       ctx,
		path:      path,
		persister: persister,
	}
}

// HandleFrame persist each frame
func (v *videocapture) handleFrame(ctx context.Context, frame *VideoFrame) error {
	file := filepath.Join(v.path, fmt.Sprintf("frame%d.jpeg",frame.Timestamp.UnixMilli()))
	if err := v.persister.Persist(ctx, file, frame.Content); err != nil {
		return fmt.Errorf("creating frame file: %w", err)
	}
	return nil
}

// Close stops the recording of the video capture
func (v *videocapture) Close() error {
	return nil
}
