// base_handler.go
package gobase

import (
	"errors"
	"strings"
	//"fmt"
	"bufio"
	"net"
	"strconv"
	"time"
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
	reader                *bufio.Reader
	writer                *bufio.Writer
	writeChan             chan []byte
	writtingLoopCloseChan chan bool
	writeEmptyChan        chan bool
	checkEmptyChan        chan bool
	closed                bool
	IBaseTCPStreamHandle
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
	c.writeEmptyChan = make(chan bool, 1000)
	c.checkEmptyChan <- true
	<-c.writeEmptyChan
	close(c.writeEmptyChan)
	c.writeEmptyChan = nil
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
		c.Conn.SetDeadline(time.Now().Add(2 * time.Minute))
	}
}

func (c *BaseTCPStream) writeLoop() {
exit1:
	for {
		select {
		case data := <-c.writeChan:
			c.write(data)
			if len(c.writeChan) == 0 && c.writeEmptyChan != nil {
				c.writeEmptyChan <- true
			}
		case <-c.checkEmptyChan:
			if len(c.writeChan) == 0 {
				c.writeEmptyChan <- true
			}
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
		if c.IBaseTCPStreamHandle != nil {
			c.IBaseTCPStreamHandle.OnException(err)
		}
	} else {
		c.writer.Flush()
		c.Conn.SetDeadline(time.Now().Add(2 * time.Minute))
	}
}

func (c *BaseTCPStream) start() {
	c.reader = bufio.NewReaderSize(c.Conn, 32*1024)
	c.writer = bufio.NewWriterSize(c.Conn, 32*1024)
	c.closed = false
	c.writeChan = make(chan []byte, 10)
	c.writtingLoopCloseChan = make(chan bool, 1)
	c.checkEmptyChan = make(chan bool, 1)
	c.Conn.SetDeadline(time.Now().Add(2 * time.Minute))
	//c.Conn.(*net.TCPConn).SetNoDelay(false)
	go c.readLoop()
	go c.writeLoop()
}

/// TCP Session
func (c *BaseTCPSession) Start() {
	c.start()
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

//addr: "127.0.0.1:80"
func (c *BaseTCPClient) ConnectByAddr(addr string) error {
	c.closed = true
	c.RemoteAddress = addr
	//阻塞
	conn, err := net.DialTimeout("tcp", addr, time.Duration(15)*time.Second)
	if err != nil {
		//log.Error("connect failed, err: ", err)
		c.IBaseTCPStreamHandle.(IBaseTCPClientHandle).OnConnect(false)
		return err
	}
	c.Conn = conn

	if _, ok := c.IBaseTCPStreamHandle.(IBaseTCPClientHandle); ok {
		c.IBaseTCPStreamHandle.(IBaseTCPClientHandle).OnConnect(true)
	}
	c.start()
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
	*net.UDPConn
	IBaseUDPStreamHandle
	closed                bool
	writeChan             chan *UDPMsg
	writtingLoopCloseChan chan bool
	writeEmptyChan        chan bool
	checkEmptyChan        chan bool
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
	s.UDPConn, err = net.ListenUDP("udp4", udpAddr)
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
	s.checkEmptyChan = make(chan bool, 1)
	go s.readLoop()
	go s.writeLoop()
end:
	return
}

func (s *BaseUDPStream) readLoop() {
	for {
		p := make([]byte, 1024)
		n, addr, err := s.UDPConn.ReadFromUDP(p)
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
			s.UDPConn.WriteTo(udpMsg.data, udpMsg.destAddr)
			if len(s.writeChan) == 0 && s.writeEmptyChan != nil {
				s.writeEmptyChan <- true
			}
		case <-s.checkEmptyChan:
			if len(s.writeChan) == 0 {
				s.writeEmptyChan <- true
			}
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
	udpMsg := &UDPMsg{
		data:     data,
		destAddr: addr,
	}
	s.writeChan <- udpMsg
}

func (s *BaseUDPStream) Flush() {
	s.writeEmptyChan = make(chan bool, 1000)
	s.checkEmptyChan <- true
	<-s.writeEmptyChan
	close(s.writeEmptyChan)
	s.writeEmptyChan = nil
}

func (s *BaseUDPStream) Close() {
	if s.closed != true {
		s.UDPConn.Close()
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
