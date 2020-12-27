package main

import (
	"fmt"
)

type SequenceNumber int

type TCPOutboundProcess struct {
	SenderBufferProcess
	DestID ProcessID
	// Unlimited size
	ToSend []TCPDataMessage
	NextSeq SequenceNumber
}

const InboundProcessBufferSize = 5

type TCPInboundProcess struct {
	ReceiverBufferProcess
	SourceID ProcessID
	// Limited to size InboundProcessBufferSize
	Received []TCPDataMessage
	NextSeq SequenceNumber
}

func (n SequenceNumber) String() string {
	return fmt.Sprintf("seq:%d", n)
}

type TCPAckMessage struct {
	seq SequenceNumber
}

func (m TCPAckMessage) String() string {
	return fmt.Sprintf("ACK(%s)", m.seq)
}

type TCPDataMessage struct {
	Message
	seq SequenceNumber
}

func (m TCPDataMessage) String() string {
	return fmt.Sprintf("DATA(%s, %s)", m.seq, m.Message)
}

func (p *TCPOutboundProcess) innerStep(send func(RoutedMessage)) {
	p.SenderBufferProcess.Step(
		func(m RoutedMessage) {
			if m.From != p.Id() || m.To != p.DestID {
				panic(fmt.Sprintf("TCP cannot send message %s", m))
			}
			p.ToSend = append(p.ToSend, TCPDataMessage{
				Message: m.Message,
				seq: p.NextSeq,
			})
			p.NextSeq += 1
		},
		func() *RoutedMessage {
			panic("inner process of TCPOutbound cannot receive messages")
		},
	)
}

func (p *TCPOutboundProcess) Step(
	send func(RoutedMessage),
	receive func() *RoutedMessage,
) {
	defer p.innerStep(send)
	for _, messageToSend := range p.ToSend {
		// TODO: keep window size and only send if in window.
		// Also track TTL.
		send(RoutedMessage{
			Message: messageToSend,
			From: p.Id(),
			To: p.DestID,
		})
	}
	received := receive()
	if received == nil {
		return
	}
	if received.To != p.Id() || received.From != p.DestID {
		panic(fmt.Sprintf("TCP incorrect To or From fields %s", received))
	}
	if ack, ok := received.Message.(TCPAckMessage); ok {
		for i := 0; i < len(p.ToSend); i++ {
			if ack.seq == p.ToSend[i].seq {
				p.ToSend = p.ToSend[i+1:]
				break
			}
		}
	} else {
		panic(fmt.Sprintf(
			"TCP unexpected message type %T %s", received.Message, received.Message,
		))
	}
}

func (p *TCPInboundProcess) innerStep(send func(RoutedMessage)) {
	p.ReceiverBufferProcess.Step(
		func(m RoutedMessage) {
			panic("inner process of TCPInbound cannot send messages")
		},
		func() *RoutedMessage {
			if len(p.Received) > 0 {
				m := &RoutedMessage{
					Message: p.Received[0].Message,
					From: p.SourceID,
					To: p.Id(),
				}
				p.Received = p.Received[1:]
				return m
			}
			return nil
		},
	)
}

func (p *TCPInboundProcess) Step(
	send func(RoutedMessage),
	receive func() *RoutedMessage,
) {
	defer p.innerStep(send)
	received := receive()
	if received == nil {
		return
	}
	if received.To != p.Id() || received.From != p.SourceID {
		panic(fmt.Sprintf("TCP incorrect To or From fields %s", received))
	}
	if data, ok := received.Message.(TCPDataMessage); ok {
		if data.seq != p.NextSeq {
			// TODO: keep a cache of data received out of order
			return
		}
		if len(p.Received) >= InboundProcessBufferSize {
			return
		}
		p.Received = append(p.Received, data)
		p.NextSeq += 1
		send(RoutedMessage{
			Message: TCPAckMessage{
				seq: data.seq,
			},
			From: p.Id(),
			To: p.SourceID,
		})
	} else {
		panic(fmt.Sprintf(
			"TCP unexpected message type %T %s", received.Message, received.Message,
		))
	}
}

type inputBuffer struct {
	input []RoutedMessage
}

func (b *inputBuffer) popInput() *RoutedMessage {
	if len(b.input) > 0 {
		m := b.input[0]
		b.input = b.input[1:]
		return &m
	}
	return nil
}

func (b *inputBuffer) pushInput(m RoutedMessage) {
	b.input = append(b.input, m)
}

type BufferProcess interface {
	Process
	popInput() *RoutedMessage
}

type ReceiverBufferProcess struct {
	ID ProcessID
	received []RoutedMessage
	inputBuffer
}

func (p *ReceiverBufferProcess) Id() ProcessID {
	return p.ID
}

func (p *ReceiverBufferProcess) Step(
	send func(RoutedMessage),
	receive func() *RoutedMessage,
) {
	received := receive()
	if received != nil {
		p.received = append(p.received, *received)
	}
}


type SenderBufferProcess struct {
	ID ProcessID
	toSend []RoutedMessage
	inputBuffer
}

func (p *SenderBufferProcess) Id() ProcessID {
	return p.ID
}

func (p *SenderBufferProcess) Step(
	send func(RoutedMessage),
	receive func() *RoutedMessage,
) {
	if len(p.toSend) > 0 {
		send(p.toSend[0])
		p.toSend = p.toSend[1:]
	}
}

type MultiTCPProcess struct {
	Process
	outboundProcs map[ProcessID]*TCPOutboundProcess
	inboundProcs map[ProcessID]*TCPInboundProcess
}

func (p *MultiTCPProcess) connect(pid ProcessID) {
	if p.outboundProcs == nil {
		p.outboundProcs = make(map[ProcessID]*TCPOutboundProcess)
		p.inboundProcs = make(map[ProcessID]*TCPInboundProcess)
	}
	p.outboundProcs[pid] = &TCPOutboundProcess{
		SenderBufferProcess: SenderBufferProcess{ID: p.Id()},
		DestID: pid,
	}
	p.inboundProcs[pid] = &TCPInboundProcess{
		ReceiverBufferProcess: ReceiverBufferProcess{ID: p.Id()},
		SourceID: pid,
	}
}

func innerTCPStep(bp BufferProcess, send func(RoutedMessage)) {
	bp.Step(
		send,
		bp.popInput,
	)
}

func (p *MultiTCPProcess) Step(
	send func(RoutedMessage),
	receive func() *RoutedMessage,
) {
	for received := receive(); received != nil; received = receive() {
		switch received.Message.(type) {
		case TCPAckMessage:
			outboundProc, ok := p.outboundProcs[received.From]
			if !ok {
				panic(fmt.Sprintf("Received ACK from %s before sending any messages", received.From))
			}
			outboundProc.SenderBufferProcess.pushInput(*received)
		case TCPDataMessage:
			if _, ok := p.inboundProcs[received.From]; !ok {
				p.connect(received.From)
			}
			p.inboundProcs[received.From].ReceiverBufferProcess.pushInput(*received)
		default:
			panic(fmt.Sprintf("Received unexpected message type %T %v", received.Message, received.Message))
		}
	}
	for _, outboundProc := range p.outboundProcs {
		innerTCPStep(outboundProc, send)
	}
	for _, inboundProc := range p.inboundProcs {
		innerTCPStep(inboundProc, send)
	}
	if p.Process == nil {
		return
	}
	p.Process.Step(
		func(m RoutedMessage) {
			if _, ok := p.outboundProcs[m.To]; !ok {
				p.connect(m.To)
			}
			sender := &p.outboundProcs[m.To].SenderBufferProcess
			sender.toSend = append(sender.toSend, m)
		},
		func() *RoutedMessage {
			for _, inboundProc := range p.inboundProcs {
				buffer := &inboundProc.ReceiverBufferProcess
				if len(buffer.received) > 0 {
					m := buffer.received[0]
					buffer.received = buffer.received[1:]
					return &m
				}
			}
			return nil
		},
	)
}
