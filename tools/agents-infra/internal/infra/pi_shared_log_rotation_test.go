//go:build !windows

package infra

import (
	"bytes"
	"errors"
	"io"
	"testing"
	"time"
)

type fakeSharedLogClock struct {
	now time.Time
}

func (clock *fakeSharedLogClock) Now() time.Time {
	return clock.now
}

func (clock *fakeSharedLogClock) advance(duration time.Duration) {
	clock.now = clock.now.Add(duration)
}

type fakeSharedLogSegment struct {
	openedAt time.Time
	data     bytes.Buffer
	closed   bool
}

func (segment *fakeSharedLogSegment) Write(data []byte) (int, error) {
	if segment.closed {
		return 0, io.ErrClosedPipe
	}
	return segment.data.Write(data)
}

func (segment *fakeSharedLogSegment) Close() error {
	segment.closed = true
	return nil
}

type fakeSharedLogSink struct {
	segments  []*fakeSharedLogSegment
	initial   []byte
	openErr   error
	rotateErr error
	pruneErr  error
}

func (sink *fakeSharedLogSink) Open(at time.Time) (io.WriteCloser, int64, error) {
	if sink.openErr != nil {
		return nil, 0, sink.openErr
	}
	segment := &fakeSharedLogSegment{openedAt: at}
	_, _ = segment.data.Write(sink.initial)
	sink.segments = append(sink.segments, segment)
	return segment, int64(len(sink.initial)), nil
}

func (sink *fakeSharedLogSink) Rotate(at time.Time) (io.WriteCloser, error) {
	if sink.rotateErr != nil {
		return nil, sink.rotateErr
	}
	segment := &fakeSharedLogSegment{openedAt: at}
	sink.segments = append(sink.segments, segment)
	return segment, nil
}

func (sink *fakeSharedLogSink) Prune(maxSegmentBytes int64, maxSegments int) error {
	if sink.pruneErr != nil {
		return sink.pruneErr
	}
	for _, segment := range sink.segments {
		if int64(segment.data.Len()) > maxSegmentBytes {
			return errors.New("managed log archive exceeds max_segment_bytes")
		}
	}
	if excess := len(sink.segments) - maxSegments; excess > 0 {
		sink.segments = append([]*fakeSharedLogSegment(nil), sink.segments[excess:]...)
	}
	return nil
}

func (sink *fakeSharedLogSink) footprint() int64 {
	var total int64
	for _, segment := range sink.segments {
		total += int64(segment.data.Len())
	}
	return total
}

// Production call site: startUnauthorizedRuntime -> openSharedRotatingLog ->
// newSharedRotatingLogWriter. The fake sink/clock exercise the same writer that
// receives both stdout and stderr from the managed runtime.
func TestSharedRuntimeLogRotatesBeforeFirstBytePastExactCap(t *testing.T) {
	clock := &fakeSharedLogClock{now: time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)}
	sink := &fakeSharedLogSink{}
	writer, err := newSharedRotatingLogWriter(sink, clock, 4, 3)
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Close()

	if n, err := writer.Write([]byte("abcd")); err != nil || n != 4 {
		t.Fatalf("exact-cap write n=%d err=%v", n, err)
	}
	if len(sink.segments) != 1 || sink.segments[0].data.String() != "abcd" {
		t.Fatalf("rotation occurred before cap: %#v", sink.segments)
	}

	clock.advance(time.Hour)
	if n, err := writer.Write([]byte("efghi")); err != nil || n != 5 {
		t.Fatalf("cross-cap write n=%d err=%v", n, err)
	}
	if len(sink.segments) != 3 {
		t.Fatalf("segments=%d want=3", len(sink.segments))
	}
	if got := []string{sink.segments[0].data.String(), sink.segments[1].data.String(), sink.segments[2].data.String()}; !equalStrings(got, []string{"abcd", "efgh", "i"}) {
		t.Fatalf("segments=%q want=[abcd efgh i]", got)
	}
	if got := sink.segments[1].openedAt; !got.Equal(clock.now) {
		t.Fatalf("rotation time=%s want=%s", got, clock.now)
	}
}

func TestSharedRuntimeLogPrunesOldestSegmentsDeterministically(t *testing.T) {
	clock := &fakeSharedLogClock{now: time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)}
	sink := &fakeSharedLogSink{}
	writer, err := newSharedRotatingLogWriter(sink, clock, 3, 2)
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Close()

	if n, err := writer.Write([]byte("abcdefghij")); err != nil || n != 10 {
		t.Fatalf("write n=%d err=%v", n, err)
	}
	if len(sink.segments) != 2 {
		t.Fatalf("segments=%d want=2", len(sink.segments))
	}
	if got := []string{sink.segments[0].data.String(), sink.segments[1].data.String()}; !equalStrings(got, []string{"ghi", "j"}) {
		t.Fatalf("retained segments=%q want=[ghi j]", got)
	}
}

func TestSharedRuntimeLogMultiDayFootprintNeverExceedsConfiguredProduct(t *testing.T) {
	const maxSegmentBytes int64 = 127
	const maxSegments = 5
	clock := &fakeSharedLogClock{now: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)}
	sink := &fakeSharedLogSink{}
	writer, err := newSharedRotatingLogWriter(sink, clock, maxSegmentBytes, maxSegments)
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Close()

	dailyOutput := bytes.Repeat([]byte("runtime-output\n"), 97)
	for day := 0; day < 45; day++ {
		clock.advance(24 * time.Hour)
		if n, err := writer.Write(dailyOutput); err != nil || n != len(dailyOutput) {
			t.Fatalf("day=%d write n=%d err=%v", day, n, err)
		}
		if got, bound := sink.footprint(), maxSegmentBytes*maxSegments; got > bound {
			t.Fatalf("day=%d footprint=%d exceeds bound=%d", day, got, bound)
		}
		if len(sink.segments) > maxSegments {
			t.Fatalf("day=%d segments=%d exceeds max=%d", day, len(sink.segments), maxSegments)
		}
		for index, segment := range sink.segments {
			if got := int64(segment.data.Len()); got > maxSegmentBytes {
				t.Fatalf("day=%d segment=%d bytes=%d exceeds cap=%d", day, index, got, maxSegmentBytes)
			}
		}
	}
}

func TestSharedRuntimeLogWriterRefusesAbsentNumericPolicy(t *testing.T) {
	clock := &fakeSharedLogClock{now: time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)}
	for _, test := range []struct {
		name            string
		maxSegmentBytes int64
		maxSegments     int
	}{
		{name: "absent max segment bytes", maxSegments: 2},
		{name: "absent max segments", maxSegmentBytes: 128},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := newSharedRotatingLogWriter(&fakeSharedLogSink{}, clock, test.maxSegmentBytes, test.maxSegments); err == nil {
				t.Fatal("rotating writer admitted absent numeric policy")
			}
		})
	}
}

func TestSharedRuntimeLogWriterRefusesMissingDependenciesAndSinkOpenFailure(t *testing.T) {
	clock := &fakeSharedLogClock{now: time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)}
	if _, err := newSharedRotatingLogWriter(nil, clock, 4, 2); err == nil {
		t.Fatal("rotating writer admitted a nil sink")
	}
	if _, err := newSharedRotatingLogWriter(&fakeSharedLogSink{}, nil, 4, 2); err == nil {
		t.Fatal("rotating writer admitted a nil clock")
	}
	want := errors.New("open failed")
	if _, err := newSharedRotatingLogWriter(&fakeSharedLogSink{openErr: want}, clock, 4, 2); !errors.Is(err, want) {
		t.Fatalf("open error=%v want=%v", err, want)
	}
}

func TestSharedRuntimeLogWriterRefusesPreexistingOversizedSegment(t *testing.T) {
	clock := &fakeSharedLogClock{now: time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)}
	sink := &fakeSharedLogSink{initial: []byte("12345")}
	if _, err := newSharedRotatingLogWriter(sink, clock, 4, 2); err == nil {
		t.Fatal("rotating writer silently admitted an already-oversized segment")
	}
	if !sink.segments[0].closed {
		t.Fatal("refused oversized segment was left open")
	}
}

func TestSharedRuntimeLogWriterPropagatesRotationAndPruningFailures(t *testing.T) {
	clock := &fakeSharedLogClock{now: time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)}
	for _, test := range []struct {
		name  string
		apply func(*fakeSharedLogSink)
	}{
		{name: "rotation", apply: func(sink *fakeSharedLogSink) { sink.rotateErr = errors.New("rotate failed") }},
		{name: "pruning", apply: func(sink *fakeSharedLogSink) { sink.pruneErr = errors.New("prune failed") }},
	} {
		t.Run(test.name, func(t *testing.T) {
			sink := &fakeSharedLogSink{}
			writer, err := newSharedRotatingLogWriter(sink, clock, 2, 2)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := writer.Write([]byte("ab")); err != nil {
				t.Fatal(err)
			}
			test.apply(sink)
			if _, err := writer.Write([]byte("c")); err == nil {
				t.Fatalf("%s failure was swallowed", test.name)
			}
		})
	}
}

func TestSharedRuntimeLogWriterCloseIsIdempotentAndRefusesLaterWrites(t *testing.T) {
	clock := &fakeSharedLogClock{now: time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)}
	writer, err := newSharedRotatingLogWriter(&fakeSharedLogSink{}, clock, 4, 2)
	if err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("second close: %v", err)
	}
	if _, err := writer.Write([]byte("x")); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("write after close error=%v want=%v", err, io.ErrClosedPipe)
	}
}
