package influxdb

import (
	"bytes"
	"code.google.com/p/goprotobuf/proto"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"protocol"
	"tcp"
)

type Client struct {
	Host     string
	Port     string
	User     string
	Password string
	Database string
	Conn     net.Conn
	Buffer   *bytes.Buffer
}

func (self *Client) handshake() error {
	self.ReadRaw()
	req := &tcp.Greeting{}
	proto.Unmarshal(self.Buffer.Bytes(), req)
	fmt.Printf("Initial Request From Server: %+v\n", req)

	// response
	resp := &tcp.Greeting{
		Account: &tcp.Account{
			Name: []byte(self.User),
			Password: []byte(self.Password),
		},
		Database: []byte(self.Database),
		Sequence: proto.Uint32(*req.Sequence + 1),
	}

	self.WriteRequest(resp)
	fmt.Printf("Initial Response From Client: %+v\n", resp)

	/// ack
	self.ReadRaw()
	ack := &tcp.Greeting{}
	proto.Unmarshal(self.Buffer.Bytes(), ack)
	fmt.Printf("Ack Response From Server: %+v\n", ack)

	if *ack.Type != tcp.Greeting_ACK {
		return errors.New("Authenticate failed")
	}

	return nil
}

func NewTcpClient(host, port, user, password, database string) (*Client, error) {
	client := &Client{host, port, user, password, database, nil, nil}
	err := client.connect("tcp", fmt.Sprintf("%s:%s", host, port))
	return client, err
}

func NewUnixClient(host, user, password, database string) (*Client, error) {
	client := &Client{host, "0", user, password, database, nil, nil}
	err := client.connect("unix", host)
	return client, err
}

func (self *Client) connect(protocol, hostspec string) (error) {
	conn, err := net.Dial(protocol, hostspec)
	if err != nil {
		return err
	}
	self.Conn = conn

	message := make([]byte, 0, 8192)
	self.Buffer = bytes.NewBuffer(message)
	return self.handshake()
}

func (self *Client) ReadRaw() error {
	var messageSizeU uint32
	var err error

	self.Buffer.Reset()

	err = binary.Read(self.Conn, binary.LittleEndian, &messageSizeU)
	if err != nil {
		return err
	}
	size := int64(messageSizeU)
	reader := io.LimitReader(self.Conn, size)
	_, err = io.Copy(self.Buffer, reader)
	if err != nil {
		return err
	}

	return nil
}

func (self *Client) WriteRequest(request interface{}) error {
	var messageSizeU uint32
	var d []byte

	if req, ok := request.(*tcp.Greeting); ok {
		d, _ = proto.Marshal(req)
	} else if req, ok := request.(*tcp.Command); ok {
		d, _ = proto.Marshal(req)
	} else {
		return errors.New(fmt.Sprintf("unsupported type %v", request))
	}

	self.Buffer.Reset()
	messageSizeU = uint32(len(d))
	binary.Write(self.Buffer, binary.LittleEndian, messageSizeU)
	_, err := io.Copy(self.Buffer, bytes.NewReader(d))
	if err != nil {
		return err
	}
	_, err = self.Conn.Write(self.Buffer.Bytes())
	if err != nil {
		return err
	}

	return nil
}

func (self *Client) Close() error {
	v := tcp.Command_CLOSE
	request := &tcp.Command{
		Type: &v,
	}

	self.WriteRequest(request)
	self.Conn.Close()
	return nil
}

func (self *Client) Query(query string) error {
	v := tcp.Command_QUERY
	request := &tcp.Command{
		Type: &v,
		Query: &tcp.Command_Query{
			Query: []byte(query),
		},
	}

	self.WriteRequest(request)

	for {
		err := self.ReadRaw()
		if err != nil {
			return err
		}
		resp := &tcp.Command{}
		err = proto.Unmarshal(self.Buffer.Bytes(), resp)
		if err != nil {
			return err
		}
		if *resp.Continue == false {
			break
		}

		fmt.Printf(".")
	}

	return nil
}

func (self *Client) WriteSeries(series *protocol.Series) error {
	v := tcp.Command_WRITESERIES
	request := &tcp.Command{
		Type: &v,
		Series: &tcp.Command_Series{
			Series: []*protocol.Series{},
		},
	}

	request.GetSeries().Series = append(request.GetSeries().Series, series)
	self.WriteRequest(request)

	err := self.ReadRaw()
	if err != nil {
		return err
	}
	resp := &tcp.Command{}
	err = proto.Unmarshal(self.Buffer.Bytes(), resp)
	if err != nil {
		return err
	}

	return nil
}
