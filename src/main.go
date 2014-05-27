package main

import (
	"code.google.com/p/goprotobuf/proto"
	"fmt"

	"protocol"
	"influxdb"
)

func main() {
	client, err := influxdb.NewUnixClient("/tmp/influxdb.sock", "root", "root", "debug")
	if err != nil {
		fmt.Printf("Error: %+v\n", err)
		return
	}
	defer client.Close()

	fmt.Printf("Client: %+v\n", client)
	//err = client.Query("select * from debug")
	//return

	for i := 0; i < 4096; i++ {
		err = client.WriteSeries(&protocol.Series{
			Name:   proto.String("debug"),
			Fields: []string{"debug", "sever", "point"},
			Points: []*protocol.Point{
				&protocol.Point{
					Values: []*protocol.FieldValue{
						&protocol.FieldValue{StringValue: proto.String("helo")},
						&protocol.FieldValue{StringValue: proto.String("main")},
						&protocol.FieldValue{DoubleValue: proto.Float64(3.5)},
					},
				},
			},
		})
		if err != nil {
			fmt.Printf("Error: %+v\n", err)
			return
		}
	}

	return

	err = client.Query("select * from debug")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}
