package internal

import (
	"log"
	"sync"
	"time"
)

//Message ...
type Message struct {
	topic         string
	content       map[string]interface{}
	contentString []string
	addedAt       int64
}

//MessageQueue  ...
type MessageQueue struct {
	agent               *Agent
	queue               []Message
	queueLock           *sync.Mutex
	lastUploadTimestamp int64
	backoffSeconds      int
}

func newMessageQueue(agent *Agent) *MessageQueue {
	mq := &MessageQueue{
		agent:               agent,
		queue:               make([]Message, 0),
		queueLock:           &sync.Mutex{},
		lastUploadTimestamp: 0,
		backoffSeconds:      0,
	}

	return mq
}

func (mq *MessageQueue) start() {
	flushTicker := time.NewTicker(1 * time.Second)

	go func() {
		defer mq.agent.recoverAndLog()

		for {
			select {
			case <-flushTicker.C:
				mq.queueLock.Lock()
				l := len(mq.queue)
				mq.queueLock.Unlock()

				// if l > 0 && (mq.lastUploadTimestamp+int64(mq.backoffSeconds) < time.Now().Unix()) {
				if l > 0 {
					mq.expire()
					mq.flush()
				}
				// }
			}
		}
	}()
}

func (mq *MessageQueue) expire() {
	now := time.Now().Unix()

	mq.queueLock.Lock()
	for i := len(mq.queue) - 1; i >= 0; i-- {
		if mq.queue[i].addedAt < now-10*60 {
			mq.queue = mq.queue[i+1:]
			break
		}
	}
	mq.queueLock.Unlock()
}

func (mq *MessageQueue) flush() {
	log.Println("Flushing the queue")
	mq.queueLock.Lock()
	outgoing := mq.queue
	mq.queue = make([]Message, 0)
	mq.queueLock.Unlock()

	messages := make([]interface{}, 0)
	payload := make([]string, 0)
	for _, m := range outgoing {
		message := map[string]interface{}{
			"topic":   m.topic,
			"content": m.content,
		}

		messages = append(messages, message)
		payload = append(payload, m.contentString...)
	}

	// for _, msg := range mq.queue {
	// 	payload = append(payload, msg.contentString...)
	// }
	// payload := map[string]interface{}{
	// 	"messages": messages,
	// }

	mq.lastUploadTimestamp = time.Now().Unix()
	log.Println("uploading the queue")
	if _, err := mq.agent.apiRequest.push("upload", payload); err == nil {
		// reset backoff
		mq.backoffSeconds = 0
	} else {
		// prepend outgoing messages back to the queue
		mq.queueLock.Lock()
		mq.queue = append(outgoing, mq.queue...)
		mq.queueLock.Unlock()

		// increase backoff up to 1 minute
		mq.agent.log("Error uploading messages to dashboard, backing off next upload")
		if mq.backoffSeconds == 0 {
			mq.backoffSeconds = 10
		} else if mq.backoffSeconds*2 < 60 {
			mq.backoffSeconds *= 2
		}

		log.Println("ERR: ", err)
	}
}
func (mq *MessageQueue) pushMessage(topic string, messages []string) {
	defer mq.flush()
	m := Message{
		topic:         topic,
		contentString: messages,
		addedAt:       time.Now().Unix(),
	}

	mq.queueLock.Lock()
	mq.queue = append(mq.queue, m)
	mq.queueLock.Unlock()

	log.Printf("Added message to the queue for topic: %v", topic)
	log.Printf("%v", messages)
}

func (mq *MessageQueue) addMessage(topic string, message map[string]interface{}) {
	m := Message{
		topic:   topic,
		content: message,
		addedAt: time.Now().Unix(),
	}

	mq.queueLock.Lock()
	mq.queue = append(mq.queue, m)
	mq.queueLock.Unlock()

	mq.agent.log("Added message to the queue for topic: %v", topic)
	mq.agent.log("%v", message)
}
