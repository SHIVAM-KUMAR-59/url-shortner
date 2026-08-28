package nodelease

import (
	"context"
	"errors"
	"fmt"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

type Leaser struct {
	client  *clientv3.Client
	leaseID clientv3.LeaseID
	nodeID  int64
}

func NewLeaser(endpoints []string) (*Leaser, error) {
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	})

	if err != nil {
		return nil, err
	}

	return &Leaser{
		client: client,
	}, nil
}

func (l *Leaser) AcquireNodeID(ctx context.Context, maxNodeID int64) (int64, error) {
	leaseResp, err := l.client.Grant(ctx, 10)
	if err != nil {
		return -1, err
	}

	var nodeId int64 = -1
	for id := int64(0); id <= maxNodeID; id++ {
		key := fmt.Sprintf("/nodeids/%d", id)

		txn := l.client.Txn(ctx)
		txnResp, err := txn.
			If(clientv3.Compare(clientv3.CreateRevision(key), "=", 0)).
			Then(clientv3.OpPut(key, "claimed", clientv3.WithLease(leaseResp.ID))).
			Commit()

		if err != nil {
			return -1, err
		}

		if txnResp.Succeeded {
			nodeId = id
			l.nodeID = id
			l.leaseID = leaseResp.ID
			break
		}

	}

	if nodeId == -1 {
		return -1, errors.New("Failed to allocate node, all node IDs are taken.")
	}

	return nodeId, nil
}

func (l *Leaser) KeepAlive(ctx context.Context) error {
	keepAliveCh, err := l.client.KeepAlive(ctx, l.leaseID)
	if err != nil {
		return err
	}

	go func() {
		for range keepAliveCh {
			// channel delivers responses on each successful renewal;
			// draining it is what keeps the lease alive
		}
	}()

	return nil
}
