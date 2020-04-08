package mixnet

import (
	"fmt"
	"log"
	"testing"
	"time"
)

func msgForId(i int) [InnerMessageLength]byte {
	var msg [InnerMessageLength]byte
	msg[0] = byte(i)
	return msg
}

func TestSmoke(t *testing.T) {
	const depth = 3
	masterKeys := make([]string, depth)
	for i := range masterKeys {
		masterKeys[i] = fmt.Sprintf("key%d", i)
	}
	addrs := make([]string, depth)
	for i := range masterKeys {
		addrs[i] = fmt.Sprintf("127.0.0.1:%d", 8000+i)
	}
	recv := make(chan string, 1)
	for i := range masterKeys {
		go func(i int) {
			msc := &MixnetServerConfig{
				MinBatch:           10,
				InputMessageLength: ForwardMessageLength(i),
			}
			if i != 0 {
				msc.NextAddr = "http://" + addrs[i-1]
			}
			ms := NewMixnetServer(msc, masterKeys[i])
			if i == 0 {
				ms.MessageHandler = func(msg []byte) {
					fmt.Printf("msg: %v\n", msg)
					recv <- string(msg)
				}
			}
			err := ms.Run(addrs[i])
			log.Fatal(err)
		}(i)
	}

	mc := MixnetClientConfig{
		Addr:    "http://" + addrs[len(addrs)-1],
		PubKeys: make([][32]byte, depth),
	}
	for i := range masterKeys {
		mc.PubKeys[i] = PubKey(masterKeys[i]) // TODO: test pubkey retrieval over http
	}

	cl := NewMixnetClient(&mc)

	const count = 10
	sent := make(map[string]bool)
	for i := 0; i < count; i++ {
		msg := msgForId(i)
		sent[string(msg[:])] = true
		err := cl.SendMessage(msg[:])
		if err != nil {
			t.Errorf("SendMessage: %s", err.Error())
		}
	}

	stop := make(chan struct{})

	go func() {
		var dummyMsg [InnerMessageLength]byte
		for {
			time.Sleep(10 * time.Millisecond)
			_ = cl.SendMessage(dummyMsg[:])
			select {
			case <-stop:
				return
			default:
			}
		}
	}()

	// TODO: do not block the last mixer while sending

	for msg := range recv {
		if len(sent) == 0 {
			break
		}
		if !sent[msg] {
			continue
		}
		delete(sent, msg)
	}
	close(stop)

}
