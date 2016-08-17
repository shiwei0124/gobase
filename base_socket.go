// base_handler.go
package gobase

import (
	"errors"
	"strings"
	"sync"
	//"fmt"
	"bufio"
	"net"
	"strconv"
	"time"
)

const (
	DEFAULT_DEADLINE        = 130
	DEFAULT_CONNECT_TIMEOUT = 15
)

type IBaseIOStream interface {
	Close()
}

type IBaseStreamHandle interface {
	OnClose()
	OnException(err error)
}

type IBaseTCPStreamHandle interface {
	IBaseStreamHandle
	OnRead(data []byte)
}

type IBaseTCPSessionHandle interface {
	IBaseTCPStreamHandle
	OnStart()
}

type IBaseTCPClientHandle interface {
	IBaseTCPStreamHandle
	OnConnect(bConnected bool)
}

type BaseIOStreamHandle struct {
}

type BaseTCPSessionHandle struct {
	BaseIOStreamHandle
}

type BaseTCPClientHandle struct {
	BaseIOStreamHandle
}

func (h *BaseIOStreamHandle) OnRead(data []byte) {
	//log.Trace("BaseIOStreamHandle onRead")
}

func (h *BaseIOStreamHandle) OnClose() {
	//log.Trace("BaseIOStreamHandle onClose")
}

func (h *BaseIOStreamHandle) OnException(err error) {
	//log.Trace("BaseIOStreamHandle onException, err: ", err)
}

func (h *BaseTCPSessionHandle) OnStart() {
	//log.Trace("BaseTCPSessionHandle onStart")
}

func (h *BaseTCPClientHandle) OnConnect(bConnected bool) {
	//log.Trace("BaseTCPClientHandle onConnect")
}

type BaseTCPStream struct {
	net.Conn
	deadLine              time.Duration //unit: second
	reader                *bufio.Reader
	writer                *bufio.Writer
	writeChan             chan []byte
	writtingLoopCloseChan chan bool
	closed                bool
	IBaseTCPStreamHandle
	writeEmptyWait *sync.WaitGroup
}

type BaseTCPSession struct {
	BaseTCPStream
}

type BaseTCPClient struct {
	BaseTCPStream
	RemoteAddress string
}

func (c *BaseTCPStream) Close() {
	if c.closed != true {
		c.Conn.Close()
		c.closed = true
		c.writtingLoopCloseChan <- true
		if c.IBaseTCPStreamHandle != nil {
			c.IBaseTCPStreamHandle.OnClose()
		}
	}
}

func (c *BaseTCPStream) Write(data []byte) {
	if c.closed != true {
		c.writeEmptyWait.Add(1)
		c.writeChan <- data
	}
}

func (c *BaseTCPStream) WriteString(data string) {
	if c.closed != true {
		dataBytes := []byte(data)
		c.Write(dataBytes)
	}
}

func (c *BaseTCPStream) Flush() {
	if c.writeEmptyWait != nil {
		c.writeEmptyWait.Wait()
	}
}

func (c *BaseTCPStream) readLoop() {
	p := make([]byte, 1024)
	for {
		n, err := c.reader.Read(p)
		if err != nil {
			if c.IBaseTCPStreamHandle != nil {
				c.IBaseTCPStreamHandle.OnException(err)
			}
			break
		} else {
			//log.Critical("read bytes num: %d", n)
		}
		if c.IBaseTCPStreamHandle != nil {
			c.IBaseTCPStreamHandle.OnRead(p[:n])
		}
		c.Conn.SetDeadline(time.Now().Add(c.deadLine * time.Second))
	}
}

func (c *BaseTCPStream) writeLoop() {
exit1:
	for {
		select {
		case data := <-c.writeChan:
			c.write(data)
		case <-c.writtingLoopCloseChan:
			//log.Trace("session writting chan stoped")
			break exit1
		}
	}
	//log.Trace("writting loop stopped..")
}

func (c *BaseTCPStream) write(data []byte) {
	//c.Conn.Write(data)
	if _, err := c.writer.Write(data); err != nil {
		c.writeEmptyWait.Done()
		if c.IBaseTCPStreamHandle != nil {
			c.IBaseTCPStreamHandle.OnException(err)
		}
	} else {
		c.writeEmptyWait.Done()
		c.writer.Flush()
		c.Conn.SetDeadline(time.Now().Add(c.deadLine * time.Second))
	}
}

func (c *BaseTCPStream) start(deadLine time.Duration) {
	c.deadLine = deadLine
	c.reader = bufio.NewReaderSize(c.Conn, 32*1024)
	c.writer = bufio.NewWriterSize(c.Conn, 32*1024)
	c.closed = false
	c.writeChan = make(chan []byte, 10)
	c.writtingLoopCloseChan = make(chan bool, 1)
	c.writeEmptyWait = &sync.WaitGroup{}
	c.Conn.SetDeadline(time.Now().Add(c.deadLine * time.Second))
	//c.Conn.(*net.TCPConn).SetNoDelay(false)
	go c.readLoop()
	go c.writeLoop()
}

/// TCP Session
func (c *BaseTCPSession) Start() {
	c.StartByDeadLine(DEFAULT_DEADLINE)
}

func (c *BaseTCPSession) StartByDeadLine(deadLine time.Duration) {
	c.start(deadLine)
	if _, ok := c.IBaseTCPStreamHandle.(IBaseTCPSessionHandle); ok {
		c.IBaseTCPStreamHandle.(IBaseTCPSessionHandle).OnStart()
	}
}

/// TCP Client
//阻塞得到结果
func (c *BaseTCPClient) Connect(ip string, port int32) error {
	addr := ip + ":" + strconv.FormatInt(int64(port), 10)
	return c.ConnectByAddr(addr)
}

func (c *BaseTCPClient) ConnectTimeOut(ip string, port int32, timeOut time.Duration, deadLine time.Duration) error {
	addr := ip + ":" + strconv.FormatInt(int64(port), 10)
	return c.ConnectByAddrTimeOut(addr, timeOut, deadLine)
}

//addr: "127.0.0.1:80"
func (c *BaseTCPClient) ConnectByAddr(addr string) error {
	return c.ConnectByAddrTimeOut(addr, DEFAULT_CONNECT_TIMEOUT, DEFAULT_DEADLINE)
}

//addr: "127.0.0.1:80"
func (c *BaseTCPClient) ConnectByAddrTimeOut(addr string, timeOut time.Duration, deadLine time.Duration) error {
	c.closed = true
	c.RemoteAddress = addr
	//阻塞
	conn, err := net.DialTimeout("tcp", addr, timeOut*time.Second)
	if err != nil {
		//log.Error("connect failed, err: ", err)
		c.IBaseTCPStreamHandle.(IBaseTCPClientHandle).OnConnect(false)
		return err
	}
	c.Conn = conn

	if _, ok := c.IBaseTCPStreamHandle.(IBaseTCPClientHandle); ok {
		c.IBaseTCPStreamHandle.(IBaseTCPClientHandle).OnConnect(true)
	}
	c.start(deadLine)
	return nil
}

////   TCP SERVER
type IBaseTCPServerHandle interface {
	IBaseStreamHandle
	OnStart()
	OnAccept(c net.Conn)
}

type BaseTCPServerHandle struct {
}

func (h *BaseTCPServerHandle) OnStart() {
	//log.Trace("BaseTCPServerHandle onStart")
}

func (h *BaseTCPServerHandle) OnAccept(c net.Conn) {
	//log.Trace("BaseTCPServerHandle onAccept")
}

func (h *BaseTCPServerHandle) OnClose() {
	//log.Trace("BaseTCPServerHandle onClose")
}

func (h *BaseTCPServerHandle) OnException(err error) {
	//log.Trace("BaseTCPServerHandle onException, err: %s", err.Error())
}

type BaseTCPServer struct {
	net.Listener
	closed bool
	IBaseTCPServerHandle
}

func (s *BaseTCPServer) StartByAddr(addr string) (err error) {
	s.Listener, err = net.Listen("tcp", addr)
	if err != nil {
		//log.Error("server bind %s failed, err: %s", addr, err.Error())
		goto end
	} else {
		//log.Info("server bind %s successed.", addr)
	}
	if s.IBaseTCPServerHandle != nil {
		s.IBaseTCPServerHandle.OnStart()
	}

	go s.acceptLoop()
end:
	return
}
func (s *BaseTCPServer) Start(ip string, port int32) error {
	addr := ip + ":" + strconv.FormatInt(int64(port), 10)
	return s.StartByAddr(addr)
}

func (s *BaseTCPServer) acceptLoop() {
	for {
		conn, err := s.Listener.Accept()
		if err != nil {
			if s.IBaseTCPServerHandle != nil {
				s.IBaseTCPServerHandle.OnException(err)
			}
			break
		}
		if s.IBaseTCPServerHandle != nil {
			s.IBaseTCPServerHandle.OnAccept(conn)
		}
	}
	return
}

func (s *BaseTCPServer) Close() {
	if s.closed != true {
		s.Listener.Close()
		s.closed = true
		if s.IBaseTCPServerHandle != nil {
			s.IBaseTCPServerHandle.OnClose()
		}
	}
}

/////////////// UDP

type IBaseUDPStreamHandle interface {
	IBaseStreamHandle
	OnStart()
	OnRead(data []byte, addr *net.UDPAddr)
}

type UDPMsg struct {
	data     []byte
	destAddr net.Addr
}

type BaseUDPStream struct {
	net.Conn
	IBaseUDPStreamHandle
	closed                bool
	writeChan             chan *UDPMsg
	writtingLoopCloseChan chan bool
	writeEmptyWait        *sync.WaitGroup
}

func (s *BaseUDPStream) StartByAddr(addr string) error {
	if addrList := strings.Split(addr, ":"); len(addrList) != 2 {
		return errors.New("err addr format")
	} else {
		strIP := addrList[0]
		strPort := addrList[1]
		if port, err := strconv.Atoi(strPort); err != nil {
			return errors.New("err addr format, err: " + err.Error())
		} else {
			return s.Start(strIP, int32(port))
		}
	}

}

func (s *BaseUDPStream) Start(ip string, port int32) (err error) {
	udpAddr := &net.UDPAddr{
		IP:   net.ParseIP(ip),
		Port: int(port),
	}
	s.Conn, err = net.ListenUDP("udp4", udpAddr)
	if err != nil {
		//log.Error("server bind %s:%d failed, err: %v", ip, port, err)
		goto end
	} else {
		//log.Infof("server bind %s:%d successed.", ip, port)
	}
	if s.IBaseUDPStreamHandle != nil {
		s.IBaseUDPStreamHandle.OnStart()
	}

	s.writeChan = make(chan *UDPMsg, 1000)
	s.writtingLoopCloseChan = make(chan bool, 1)
	s.writeEmptyWait = &sync.WaitGroup{}
	go s.readLoop()
	go s.writeLoop()
end:
	return
}

func (s *BaseUDPStream) readLoop() {
	for {
		p := make([]byte, 1024)
		n, addr, err := s.Conn.(*net.UDPConn).ReadFromUDP(p)
		if err != nil {
			if s.IBaseUDPStreamHandle != nil {
				s.IBaseUDPStreamHandle.OnException(err)
			}
			break
		}
		if s.IBaseUDPStreamHandle != nil {
			s.IBaseUDPStreamHandle.OnRead(p[:n], addr)
		}
	}
}

//udp的底层write有锁
func (s *BaseUDPStream) writeLoop() {
exit1:
	for {
		select {
		case udpMsg := <-s.writeChan:
			s.Conn.(*net.UDPConn).WriteTo(udpMsg.data, udpMsg.destAddr)
			s.writeEmptyWait.Done()
		case <-s.writtingLoopCloseChan:
			//log.Trace("session writting chan stoped")
			break exit1
		}
	}
	//log.Trace("writting loop stopped..")
}

func (s *BaseUDPStream) WriteToUDP(data []byte, addr *net.UDPAddr) {
	s.WriteTo(data, addr)
}

func (s *BaseUDPStream) WriteTo(data []byte, addr net.Addr) {
	s.writeEmptyWait.Add(1)
	udpMsg := &UDPMsg{
		data:     data,
		destAddr: addr,
	}
	s.writeChan <- udpMsg
}

func (s *BaseUDPStream) Flush() {
	if s.writeEmptyWait != nil {
		s.writeEmptyWait.Wait()
	}
}

func (s *BaseUDPStream) Close() {
	if s.closed != true {
		s.Conn.Close()
		s.closed = true
		s.writtingLoopCloseChan <- true

		if s.IBaseUDPStreamHandle != nil {
			s.IBaseUDPStreamHandle.OnClose()
		}
	}
}

///  UDP Client
type IBaseUDPClientHandle interface {
	IBaseUDPStreamHandle
}

type BaseUDPClientHandle struct {
}

func (h *BaseUDPClientHandle) OnStart() {
	//log.Trace("BaseUDPClientHandle onStart")
}

func (h *BaseUDPClientHandle) OnException(err error) {
	//log.Trace("BaseUDPClientHandle onException, err: %s", err.Error())
}

func (h *BaseUDPClientHandle) OnClose() {
	//log.Trace("BaseUDPClientHandle onClose")
}

type BaseUDPClient struct {
	BaseUDPStream
}

///   UDP Server
type IBaseUDPServerHandle interface {
	IBaseUDPStreamHandle
}

type BaseUDPServerHandle struct {
}

func (h *BaseUDPServerHandle) OnStart() {
	//log.Trace("BaseUDPServerHandle onStart")
}

func (h *BaseUDPServerHandle) OnException(err error) {
	//log.Trace("BaseUDPServerHandle onException, err: %s", err.Error() )
}

func (h *BaseUDPServerHandle) OnClose() {
	//log.Trace("BaseUDPServerHandle onClose")
}

type BaseUDPServer struct {
	BaseUDPStream
}

/// Unix Socket
type IBaseUnixStreamHandle interface {
	IBaseStreamHandle
	OnRead(data []byte)
}

type IBaseUnixSessionHandle interface {
	IBaseUnixStreamHandle
	OnStart()
}

type IBaseUnixClientHandle interface {
	IBaseUnixStreamHandle
	OnConnect(bConnected bool)
}

type BaseUnixSessionHandle struct {
	BaseIOStreamHandle
}

type BaseUnixClientHandle struct {
	BaseIOStreamHandle
}

func (h *BaseUnixSessionHandle) OnStart() {
	//log.Trace("BaseUnixSessionHandle onStart")
}

func (h *BaseUnixClientHandle) OnConnect(bConnected bool) {
	//log.Trace("BaseUnixClientHandle onConnect")
}

type BaseUnixStream struct {
	net.Conn
	deadLine              time.Duration //unit: second
	reader                *bufio.Reader
	writer                *bufio.Writer
	writeChan             chan []byte
	writtingLoopCloseChan chan bool
	closed                bool
	IBaseUnixStreamHandle
	writeEmptyWait *sync.WaitGroup
}

type BaseUnixSession struct {
	BaseUnixStream
}

type BaseUnixClient struct {
	BaseUnixStream
	RemoteAddress string
}

func (c *BaseUnixStream) Close() {
	if c.closed != true {
		c.Conn.Close()
		c.closed = true
		c.writtingLoopCloseChan <- true
		if c.IBaseUnixStreamHandle != nil {
			c.IBaseUnixStreamHandle.OnClose()
		}
	}
}

func (c *BaseUnixStream) Write(data []byte) {
	if c.closed != true {
		c.writeEmptyWait.Add(1)
		c.writeChan <- data
	}
}

func (c *BaseUnixStream) WriteString(data string) {
	if c.closed != true {
		dataBytes := []byte(data)
		c.Write(dataBytes)
	}
}

func (c *BaseUnixStream) Flush() {
	if c.writeEmptyWait != nil {
		c.writeEmptyWait.Wait()
	}
}

func (c *BaseUnixStream) readLoop() {
	p := make([]byte, 1024)
	for {
		n, err := c.reader.Read(p)
		if err != nil {
			if c.IBaseUnixStreamHandle != nil {
				c.IBaseUnixStreamHandle.OnException(err)
			}
			break
		} else {
			//log.Critical("read bytes num: %d", n)
		}
		if c.IBaseUnixStreamHandle != nil {
			c.IBaseUnixStreamHandle.OnRead(p[:n])
		}
		c.Conn.SetDeadline(time.Now().Add(c.deadLine * time.Second))
	}
}

func (c *BaseUnixStream) writeLoop() {
exit1:
	for {
		select {
		case data := <-c.writeChan:
			c.write(data)
		case <-c.writtingLoopCloseChan:
			//log.Trace("session writting chan stoped")
			break exit1
		}
	}
	//log.Trace("writting loop stopped..")
}

func (c *BaseUnixStream) write(data []byte) {
	//c.Conn.Write(data)
	if _, err := c.writer.Write(data); err != nil {
		c.writeEmptyWait.Done()
		if c.IBaseUnixStreamHandle != nil {
			c.IBaseUnixStreamHandle.OnException(err)
		}
	} else {
		c.writeEmptyWait.Done()
		c.writer.Flush()
		c.Conn.SetDeadline(time.Now().Add(c.deadLine * time.Second))
	}
}

func (c *BaseUnixStream) start(deadLine time.Duration) {
	c.deadLine = deadLine
	c.reader = bufio.NewReaderSize(c.Conn, 32*1024)
	c.writer = bufio.NewWriterSize(c.Conn, 32*1024)
	c.closed = false
	c.writeChan = make(chan []byte, 10)
	c.writtingLoopCloseChan = make(chan bool, 1)
	c.writeEmptyWait = &sync.WaitGroup{}
	c.Conn.SetDeadline(time.Now().Add(c.deadLine * time.Second))
	//c.Conn.(*net.TCPConn).SetNoDelay(false)
	go c.readLoop()
	go c.writeLoop()
}

/// UnixSock Session
func (c *BaseUnixSession) Start() {
	c.StartByDeadLine(DEFAULT_DEADLINE)
}

func (c *BaseUnixSession) StartByDeadLine(deadLine time.Duration) {
	c.start(deadLine)
	if _, ok := c.IBaseUnixStreamHandle.(IBaseUnixSessionHandle); ok {
		c.IBaseUnixStreamHandle.(IBaseUnixSessionHandle).OnStart()
	}
}

/// UnixSock Client
//阻塞得到结果
//addr: "/tmp/fileserver.socket"
func (c *BaseUnixClient) ConnectByAddr(addr string) error {
	return c.ConnectByAddrWithDeadLine(addr, DEFAULT_DEADLINE)
}

func (c *BaseUnixClient) ConnectByAddrWithDeadLine(addr string, deadLine time.Duration) error {
	c.closed = true
	c.RemoteAddress = addr
	unixAddr, err := net.ResolveUnixAddr("unix", addr)
	if err != nil {
		c.IBaseUnixStreamHandle.(IBaseUnixClientHandle).OnException(err)
		return err
	}

	//阻塞
	conn, err := net.DialUnix("unix", nil, unixAddr)
	if err != nil {
		//log.Error("connect failed, err: ", err)
		c.IBaseUnixStreamHandle.(IBaseUnixClientHandle).OnConnect(false)
		return err
	}
	c.Conn = conn

	if _, ok := c.IBaseUnixStreamHandle.(IBaseUnixClientHandle); ok {
		c.IBaseUnixStreamHandle.(IBaseUnixClientHandle).OnConnect(true)
	}
	c.start(deadLine)
	return nil
}

////   UNIXSOCK STREAM SERVER
type IBaseUnixServerHandle interface {
	IBaseStreamHandle
	OnStart()
	OnAccept(c net.Conn)
}

type BaseUnixServerHandle struct {
}

func (h *BaseUnixServerHandle) OnStart() {
	//log.Trace("BaseUnixServerHandle onStart")
}

func (h *BaseUnixServerHandle) OnAccept(c net.Conn) {
	//log.Trace("BaseUnixServerHandle onAccept")
}

func (h *BaseUnixServerHandle) OnClose() {
	//log.Trace("BaseUnixServerHandle onClose")
}

func (h *BaseUnixServerHandle) OnException(err error) {
	//log.Trace("BaseUnixServerHandle onException, err: %s", err.Error())
}

type BaseUnixServer struct {
	net.Listener
	closed bool
	IBaseUnixServerHandle
}

func (s *BaseUnixServer) StartByAddr(addr string) (err error) {
	var unixAddr *net.UnixAddr
	unixAddr, err = net.ResolveUnixAddr("unix", addr)
	if err != nil {
		goto end
	}
	s.Listener, err = net.ListenUnix("unix", unixAddr)
	if err != nil {
		//log.Error("server bind %s failed, err: %s", addr, err.Error())
		goto end
	} else {
		//log.Info("server bind %s successed.", addr)
	}
	if s.IBaseUnixServerHandle != nil {
		s.IBaseUnixServerHandle.OnStart()
	}

	go s.acceptLoop()
end:
	return
}

func (s *BaseUnixServer) acceptLoop() {
	for {
		conn, err := s.Listener.(*net.UnixListener).AcceptUnix()
		if err != nil {
			if s.IBaseUnixServerHandle != nil {
				s.IBaseUnixServerHandle.OnException(err)
			}
			break
		}
		if s.IBaseUnixServerHandle != nil {
			s.IBaseUnixServerHandle.OnAccept(conn)
		}
	}
	return
}

func (s *BaseUnixServer) Close() {
	if s.closed != true {
		s.Listener.Close()
		s.closed = true
		if s.IBaseUnixServerHandle != nil {
			s.IBaseUnixServerHandle.OnClose()
		}
	}
}
