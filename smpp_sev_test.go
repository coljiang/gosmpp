package gosmpp

import (
	"fmt"
	"github.com/linxGnu/gosmpp/pdu"
	"net"
	"testing"
	"time"
)

func TestSevSmpp(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:2775")
	if err != nil {
		t.Fatalf("net has err - %+v", err)
	}
	//if l, err = net.Listen("tcp6", "[::1]:2775"); err != nil {
	//	panic(fmt.Sprintf("smpptest: failed to listen on a port: %v", err))
	//}
	for {
		cli, err := l.Accept()
		if err != nil {
			t.Fatal(err)
		}
		go func() {
			handleSmpp(cli)
		}()
	}
}

func handleSmpp(conn net.Conn) {
	sevConnect := NewSevConnector(conn)
	sevConnect.SetUserCheck(func(username, password, ip string) bool {
		return true
	}).SetBindingType(pdu.Transmitter)

	transmitter, err := NewSession(
		sevConnect,
		Settings{
			ReadTimeout: 2 * time.Second,

			OnPDU: func(p pdu.PDU, _ bool) {
				fmt.Printf("%+v\n", p)
			},

			OnSubmitError: func(_ pdu.PDU, err error) {
				fmt.Print(err)
			},

			OnRebindingError: func(err error) {
				fmt.Print(err)
			},

			OnClosed: func(state State) {
				fmt.Print(state)
			},
		}, -1)
	if err != nil {
		fmt.Printf("smpp has err")
	}
	fmt.Printf("%+v", transmitter)
}
