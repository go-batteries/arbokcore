package supervisors

import (
	"arbokcore/pkg/queuer"
	"arbokcore/pkg/rho"
	"arbokcore/pkg/workerpool"
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func Execute(ctx context.Context, data []*queuer.Payload) []*queuer.Payload {
	return data
}

type MockSuperVisor struct {
	demand chan int
}

func (slf *MockSuperVisor) Produce(ctx context.Context) chan []*queuer.Payload {
	resultsCh := make(chan []*queuer.Payload, 1)

	go func() {
		defer close(resultsCh)

		for {
			select {

			case d := <-slf.demand:
				results := []*queuer.Payload{}

				for i := 0; i < d; i++ {
					payload := &queuer.Payload{
						Message: []byte(fmt.Sprintf("%d", i)),
					}
					results = append(results, payload)
				}

				//Mock sleep to give other workers
				// a chance to pickup produced data
				time.Sleep(2 * time.Second)
				resultsCh <- results

			case <-ctx.Done():
				return
			}
		}
	}()

	return resultsCh
}

func (slf *MockSuperVisor) Demand(val int) {
	slf.demand <- val
}

func Test_Supervisor(t *testing.T) {
	ctx := context.Background()
	pool := workerpool.NewWorkerPool(2, Execute)
	outChan := make(chan []*queuer.Payload, 1)
	demand := make(chan int, 1)

	ms := MockSuperVisor{demand: demand}

	receivCh := ms.Produce(ctx)
	go workerpool.Dispatch(ctx, pool, receivCh)

	pool.Start(ctx)

	workerpool.Merge(ctx, pool.ResultChs, outChan)

	ms.Demand(10)
	ms.Demand(10)

	payloads := <-outChan
	payloads = append(payloads, <-outChan...)

	pool.Stop(ctx)

	msg := rho.Map(
		payloads,
		func(payload *queuer.Payload, _ int) string {
			return string(payload.Message)
		})

	expected := []string{}
	for i := 0; i < 10; i++ {
		expected = append(expected, fmt.Sprintf("%d", i))
	}

	expected = append(expected, expected...)

	require.Equal(t, expected, msg)
}
