// Package config
package config

import (
	"os"
	"strings"

	"github.com/segmentio/kafka-go"
)

var KafkaBrokerUrls = []string{"localhost:9092", "localhost:9093", "localhost:9094"}

func getKafkaBrokerURLs() []string {
	brokers := os.Getenv("KAFKA_BROKERS")
	if brokers == "" {
		brokers = "localhost:9092, localhost:9093, localhost:9094"
	}
	return strings.Split(brokers, ",")
}
func NewKafkaWrite(topic string) *kafka.Writer {
	return &kafka.Writer{
		//Addr:                   kafka.TCP(KafkaBrokerUrls...),
		Addr:                   kafka.TCP(getKafkaBrokerURLs()...),
		Topic:                  topic,
		Balancer:               &kafka.CRC32Balancer{},
		AllowAutoTopicCreation: true,
	}
}
