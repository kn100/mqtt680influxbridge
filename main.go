package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	influx "github.com/influxdata/influxdb1-client/v2"
)

func main() {
	var (
		mqttAddress     = envString("MQTT_ADDRESS", "192.168.0.17")
		mqttPort        = envString("MQTT_ADDRESS", "1883")
		mqttTopic       = envString("MQTT_TOPIC", "bme680/+")
		influxDbAddress = envString("INFLUXDB_ADDRESS", "192.168.0.17")
		influxDbDb      = envString("INFLUXDB_DB", "bme680")
	)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	influxHost := fmt.Sprintf("http://%s:%d", influxDbAddress, 8086)

	influxClient, err := influx.NewHTTPClient(influx.HTTPConfig{Addr: influxHost, Timeout: 5 * time.Second})
	if err != nil {
		log.Fatal(err)
	}

	opts := mqtt.NewClientOptions().AddBroker(fmt.Sprintf("tcp://%s:%s", mqttAddress, mqttPort))
	mqttClient := mqtt.NewClient(opts)
	if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	var f mqtt.MessageHandler = func(client mqtt.Client, message mqtt.Message) {
		log.Printf("Received message on topic: %s\nMessage: %s\n", message.Topic(), message.Payload())
		topic := message.Topic()
		val, err := strconv.ParseFloat(string(message.Payload()), 32)
		if err != nil {
			log.Println("Invalid point data, ignoring")
			return
		}
		fields := map[string]interface{}{
			topic: val,
		}
		influxPoint, err := influx.NewPoint(message.Topic(), nil, fields, time.Now())

		bp, err := influx.NewBatchPoints(influx.BatchPointsConfig{
			Database:        influxDbDb,
			Precision:       "s",
			RetentionPolicy: "30_days",
		})

		if err != nil {
			log.Fatalln("Error: ", err)
		}

		bp.AddPoint(influxPoint)
		err = influxClient.Write(bp)
		if err != nil {
			log.Println("Couldn't write to influx for some reason. Ignoring.", err)
			return
		}
		log.Println("supposedly written to influx")
	}

	if token := mqttClient.Subscribe(mqttTopic, 0, f); token.Wait() && token.Error() != nil {
		log.Fatal(token.Error())
	}
	<-sigs
	log.Println("Exiting")
}

func envString(key, fallback string) string {
	if s := os.Getenv(key); s != "" {
		return s
	}

	return fallback
}
