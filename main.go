package main

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	influx "github.com/influxdata/influxdb1-client"
)

func main() {
	var (
		mqttAddress     = envString("MQTT_ADDRESS", "192.168.0.17")
		mqttPort        = envString("MQTT_ADDRESS", "1883")
		mqttTopic       = envString("MQTT_TOPIC", "bme680/+")
		mqttClientID    = envString("MQTT_CLIENT_ID", "BME680BRIDGE")
		influxDbAddress = envString("INFLUXDB_ADDRESS", "192.168.0.17")
		influxDbDb      = envString("INFLUXDB_DB", "bme680")
	)
	fmt.Println(mqttAddress, mqttPort, mqttTopic, mqttClientID, influxDbAddress, influxDbDb)

	host, err := url.Parse(fmt.Sprintf("http://%s:%d", influxDbAddress, 8086))
	if err != nil {
		log.Fatal(err)
	}
	con, err := influx.NewClient(influx.Config{URL: *host})
	if err != nil {
		log.Fatal(err)
	}

	opts := mqtt.NewClientOptions().AddBroker(fmt.Sprintf("tcp://%s:%s", mqttAddress, mqttPort))
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	var f mqtt.MessageHandler = func(client mqtt.Client, message mqtt.Message) {
		fmt.Printf("Received message on topic: %s\nMessage: %s\n", message.Topic(), message.Payload())
		topic := message.Topic()
		val, err := strconv.ParseFloat(string(message.Payload()), 32)
		if err != nil {
			log.Println("Invalid point data, ignoring")
			return
		}

		var influxPoint = influx.Point{
			Measurement: message.Topic(),
			Time:        time.Now(),
			Fields: map[string]interface{}{
				topic: val,
			},
		}
		pts := make([]influx.Point, 1)
		pts[0] = influxPoint

		_, err = con.Write(influx.BatchPoints{
			Points:          pts,
			Database:        influxDbDb,
			RetentionPolicy: "30_days",
		})
		if err != nil {
			log.Println("Couldn't write to influx for some reason. Ignoring.", err)
			return
		}
		log.Println("supposedly written to influx")
	}

	if token := client.Subscribe(mqttTopic, 0, f); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
	}
	time.Sleep(30000000 * time.Second)
}

func envString(key, fallback string) string {
	if s := os.Getenv(key); s != "" {
		return s
	}

	return fallback
}
