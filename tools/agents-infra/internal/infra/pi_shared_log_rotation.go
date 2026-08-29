package infra

import (
	"errors"
	"io"
	"sync"
	"time"
)

type sharedLogClock interface {
	Now() time.Time
}

type sharedLogSegmentSink interface {
	Open(time.Time) (io.WriteCloser, int64, error)
	Rotate(time.Time) (io.WriteCloser, error)
	Prune(int64, int) error
}

type sharedRotatingLogWriter struct {
	mu              sync.Mutex
	sink            sharedLogSegmentSink
	clock           sharedLogClock
	current         io.WriteCloser
	currentBytes    int64
	maxSegmentBytes int64
	maxSegments     int
	closed          bool
}

func newSharedRotatingLogWriter(sink sharedLogSegmentSink, clock sharedLogClock, maxSegmentBytes int64, maxSegments int) (*sharedRotatingLogWriter, error) {
	if sink == nil {
		return nil, errors.New("shared runtime log sink is required")
	}
	if clock == nil {
		return nil, errors.New("shared runtime log clock is required")
	}
	if maxSegmentBytes <= 0 {
		return nil, errors.New("shared runtime log max_segment_bytes must be positive")
	}
	if maxSegments <= 0 {
		return nil, errors.New("shared runtime log max_segments must be positive")
	}
	current, currentBytes, err := sink.Open(clock.Now().UTC())
	if err != nil {
		return nil, err
	}
	if currentBytes < 0 || currentBytes > maxSegmentBytes {
		_ = current.Close()
		return nil, errors.New("existing shared runtime log segment exceeds max_segment_bytes")
	}
	if err := sink.Prune(maxSegmentBytes, maxSegments); err != nil {
		_ = current.Close()
		return nil, err
	}
	return &sharedRotatingLogWriter{
		sink:            sink,
		clock:           clock,
		current:         current,
		currentBytes:    currentBytes,
		maxSegmentBytes: maxSegmentBytes,
		maxSegments:     maxSegments,
	}, nil
}

func (writer *sharedRotatingLogWriter) Write(data []byte) (int, error) {
	writer.mu.Lock()
	defer writer.mu.Unlock()
	if writer.closed {
		return 0, io.ErrClosedPipe
	}

	written := 0
	for len(data) > 0 {
		if writer.currentBytes == writer.maxSegmentBytes {
			if err := writer.rotate(); err != nil {
				return written, err
			}
		}
		remaining := writer.maxSegmentBytes - writer.currentBytes
		chunkBytes := int64(len(data))
		if chunkBytes > remaining {
			chunkBytes = remaining
		}
		chunk := data[:int(chunkBytes)]
		n, err := writer.current.Write(chunk)
		written += n
		writer.currentBytes += int64(n)
		data = data[n:]
		if err != nil {
			return written, err
		}
		if n != len(chunk) {
			return written, io.ErrShortWrite
		}
	}
	return written, nil
}

func (writer *sharedRotatingLogWriter) rotate() error {
	if err := writer.current.Close(); err != nil {
		return err
	}
	current, err := writer.sink.Rotate(writer.clock.Now().UTC())
	if err != nil {
		return err
	}
	writer.current = current
	writer.currentBytes = 0
	if err := writer.sink.Prune(writer.maxSegmentBytes, writer.maxSegments); err != nil {
		_ = writer.current.Close()
		return err
	}
	return nil
}

func (writer *sharedRotatingLogWriter) Close() error {
	writer.mu.Lock()
	defer writer.mu.Unlock()
	if writer.closed {
		return nil
	}
	writer.closed = true
	return writer.current.Close()
}

type sharedSystemLogClock struct{}

func (sharedSystemLogClock) Now() time.Time {
	return time.Now()
}
