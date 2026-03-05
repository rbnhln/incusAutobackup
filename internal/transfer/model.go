package transfer

import (
	"context"
	"io"
	"time"
)

type Kind string

const (
	KindInstance Kind = "instance"
	KindVolume   Kind = "volume"
)

// RecoveryPoint is genric point
// in case of incus(os)-target: a snapshot
// in case of for example s3: a file

type RecoveryPoint struct {
	Kind      Kind
	Project   string // optional
	Subject   string // instance or volume name
	Name      string // name of the snapshot
	CreatedAt time.Time
}

// streamable object atrifact
type Artifact struct {
	Point RecoveryPoint
	Open  func(ctx context.Context) (io.ReadCloser, error)
	Size  int64
}
