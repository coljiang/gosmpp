package gosmpp

import (
	"fmt"
	"net"

	"github.com/coljiang/gosmpp/data"
	"github.com/coljiang/gosmpp/pdu"
)

var (
	// NonTLSDialer is non-tls connection dialer.
	NonTLSDialer = func(addr string) (net.Conn, error) {
		return net.Dial("tcp", addr)
	}
)

// Dialer is connection dialer.
type Dialer func(addr string) (net.Conn, error)

// Auth represents basic authentication to SMSC.
type Auth struct {
	// SMSC is SMSC address.
	SMSC       string
	SystemID   string
	Password   string
	SystemType string
}

type BindError struct {
	CommandStatus data.CommandStatusType
}

func (err BindError) Error() string {
	return fmt.Sprintf("binding error (%s): %s", err.CommandStatus, err.CommandStatus.Desc())
}

func newBindRequest(s Auth, bindingType pdu.BindingType, addressRange pdu.AddressRange) (bindReq *pdu.BindRequest) {
	bindReq = pdu.NewBindRequest(bindingType)
	bindReq.SystemID = s.SystemID
	bindReq.Password = s.Password
	bindReq.SystemType = s.SystemType
	bindReq.AddressRange = addressRange
	return
}

// Connector is connection factory interface.
type Connector interface {
	Connect() (conn *Connection, err error)
	GetBindType() pdu.BindingType
}

type connector struct {
	dialer       Dialer
	auth         Auth
	bindingType  pdu.BindingType
	addressRange pdu.AddressRange
}

func (c *connector) GetBindType() pdu.BindingType {
	return c.bindingType
}

func (c *connector) Connect() (conn *Connection, err error) {
	conn, err = connect(c.dialer, c.auth.SMSC, newBindRequest(c.auth, c.bindingType, c.addressRange))
	return
}

func connect(dialer Dialer, addr string, bindReq *pdu.BindRequest) (c *Connection, err error) {
	conn, err := dialer(addr)
	if err != nil {
		return
	}

	// create wrapped connection
	c = NewConnection(conn)

	// send binding request
	_, err = c.WritePDU(bindReq)
	if err != nil {
		_ = conn.Close()
		return
	}

	// catching response
	var (
		p    pdu.PDU
		resp *pdu.BindResp
	)

	for {
		if p, err = pdu.Parse(c); err != nil {
			_ = conn.Close()
			return
		}

		if pd, ok := p.(*pdu.BindResp); ok {
			resp = pd
			break
		}
	}

	if resp.CommandStatus != data.ESME_ROK {
		err = BindError{CommandStatus: resp.CommandStatus}
		_ = conn.Close()
	} else {
		c.systemID = resp.SystemID
	}

	return
}

// TXConnector returns a Transmitter (TX) connector.
func TXConnector(dialer Dialer, auth Auth) Connector {
	return &connector{
		dialer:      dialer,
		auth:        auth,
		bindingType: pdu.Transmitter,
	}
}

// RXConnector returns a Receiver (RX) connector.
func RXConnector(dialer Dialer, auth Auth, opts ...connectorOption) Connector {
	c := &connector{
		dialer:      dialer,
		auth:        auth,
		bindingType: pdu.Receiver,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// TRXConnector returns a Transceiver (TRX) connector.
func TRXConnector(dialer Dialer, auth Auth, opts ...connectorOption) Connector {
	c := &connector{
		dialer:      dialer,
		auth:        auth,
		bindingType: pdu.Transceiver,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// SevConnector
func SevConnector(dialer Dialer, auth Auth, opts ...connectorOption) Connector {
	c := &connector{
		dialer:      dialer,
		auth:        auth,
		bindingType: pdu.Transceiver,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

type connectorOption func(c *connector)

func WithAddressRange(addressRange pdu.AddressRange) connectorOption {
	return func(c *connector) {
		c.addressRange = addressRange
	}
}

func NewSevConnector(conn net.Conn) *sevConnector {
	return &sevConnector{conn: conn}
}

type sevConnector struct {
	//dialer       Dialer
	auth        Auth
	bindingType pdu.BindingType
	//addressRange pdu.AddressRange
	userCheck func(string, string, string) bool
	conn      net.Conn
}

func (sc *sevConnector) SetBindingType(pdu.BindingType) *sevConnector {
	return sc
}

func (sc *sevConnector) SetUserCheck(userCheckFunc func(username, password, ip string) bool) *sevConnector {
	if userCheckFunc == nil {
		return sc
	}
	sc.userCheck = userCheckFunc
	return sc
}

func (sc *sevConnector) GetBindType() pdu.BindingType {
	return sc.bindingType
}

func (sc *sevConnector) Connect() (c *Connection, err error) {
	c = NewConnection(sc.conn)
	var (
		p   pdu.PDU
		req *pdu.BindRequest
	)
	remoteIp := sc.conn.RemoteAddr().String()
	for {
		if p, err = pdu.Parse(c); err != nil {
			_ = sc.conn.Close()
			return
		}
		if pd, ok := p.(*pdu.BindRequest); ok {
			req = pd
			break
		}
	}
	// 检查用户名和密码
	if !sc.userCheck(req.SystemID, req.Password, remoteIp) {
		// 认证失败，返回 Bind Response 失败 PDU
		resp := pdu.NewBindResp(*req)
		resp.Header.CommandStatus = data.ESME_RBINDFAIL // 绑定失败状态码
		_, _ = c.WritePDU(resp)                         // 发送响应 PDU
		_ = sc.conn.Close()                             // 关闭连接
		return nil, fmt.Errorf("authentication failed for SystemID: %s", req.SystemID)
	}

	// 认证成功，返回成功 PDU
	resp := pdu.NewBindResp(*req)
	resp.Header.CommandStatus = data.ESME_ROK // 认证成功
	_, err = c.WritePDU(resp)
	if err != nil {
		return nil, err
	}

	return c, nil
}
