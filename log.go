package kapacitor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/influxdata/kapacitor/edge"
	"github.com/influxdata/kapacitor/pipeline"
	"github.com/influxdata/wlog"
)

type LogNode struct {
	node

	key string
	buf bytes.Buffer
	enc *json.Encoder

	batchBuffer *edge.BatchBuffer
}

// Create a new  LogNode which logs all data it receives
func newLogNode(et *ExecutingTask, n *pipeline.LogNode, d NodeDiagnostic) (*LogNode, error) {
	level, ok := wlog.StringToLevel[strings.ToUpper(n.Level)]
	if !ok {
		return nil, fmt.Errorf("invalid log level %s", n.Level)
	}
	nn := &LogNode{
		node:        node{Node: n, et: et, diag: d},
		key:         fmt.Sprintf("%c! %s", wlog.ReverseLevels[level], n.Prefix),
		batchBuffer: new(edge.BatchBuffer),
	}
	nn.enc = json.NewEncoder(&nn.buf)
	nn.node.runF = nn.runLog
	return nn, nil
}

func (n *LogNode) runLog([]byte) error {
	consumer := edge.NewConsumerWithReceiver(
		n.ins[0],
		edge.NewReceiverFromForwardReceiverWithStats(
			n.outs,
			edge.NewTimedForwardReceiver(n.timer, n),
		),
	)
	return consumer.Consume()

}

func (n *LogNode) BeginBatch(begin edge.BeginBatchMessage) (edge.Message, error) {
	return nil, n.batchBuffer.BeginBatch(begin)
}

func (n *LogNode) BatchPoint(bp edge.BatchPointMessage) (edge.Message, error) {
	return nil, n.batchBuffer.BatchPoint(bp)
}

func (n *LogNode) EndBatch(end edge.EndBatchMessage) (edge.Message, error) {
	return n.BufferedBatch(n.batchBuffer.BufferedBatchMessage(end))
}

func (n *LogNode) BufferedBatch(batch edge.BufferedBatchMessage) (edge.Message, error) {
	n.buf.Reset()
	if err := n.enc.Encode(batch); err != nil {
		n.incrementErrorCount()
		n.diag.Error("failed to encode batch data", err)
		return batch, nil
	}
	// TODO: fix prefix and other loger here
	n.diag.LogData(n.key, "Prefix", n.buf.String())
	return batch, nil
}

func (n *LogNode) Point(p edge.PointMessage) (edge.Message, error) {
	n.buf.Reset()
	if err := n.enc.Encode(p); err != nil {
		n.incrementErrorCount()
		n.diag.Error("failed to encode stream data", err)
		return p, nil
	}
	n.diag.LogData(n.key, "Prefix", n.buf.String())
	return p, nil
}

func (n *LogNode) Barrier(b edge.BarrierMessage) (edge.Message, error) {
	return b, nil
}
func (n *LogNode) DeleteGroup(d edge.DeleteGroupMessage) (edge.Message, error) {
	return d, nil
}
