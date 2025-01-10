package gorm

import (
	"fmt"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	uuid "github.com/satori/go.uuid"
)

var (
	maxOpenConnectionsGaugeVec = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "gorm_dbstats_max_open_connections",
		Help: "Maximum number of open connections to the database.",
	}, []string{"type", "name", "host", "db_name"})
	openConnectionsGaugeVec = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "gorm_dbstats_open_connections",
		Help: "The number of established connections both in use and idle.",
	}, []string{"type", "name", "host", "db_name"})
	inUseGaugeVec = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "gorm_dbstats_in_use",
		Help: "The number of connections currently in use.",
	}, []string{"type", "name", "host", "db_name"})
	idleGaugeVec = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "gorm_dbstats_idle",
		Help: "The number of idle connections.",
	}, []string{"type", "name", "host", "db_name"})
	waitCountGaugeVec = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "gorm_dbstats_wait_count",
		Help: "The total number of connections waited for.",
	}, []string{"type", "name", "host", "db_name"})
	waitDurationGaugeVec = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "gorm_dbstats_wait_duration",
		Help: "The total time blocked waiting for a new connection.",
	}, []string{"type", "name", "host", "db_name"})
	maxIdleClosedGaugeVec = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "gorm_dbstats_max_idle_closed",
		Help: "The total number of connections closed due to SetMaxIdleConns.",
	}, []string{"type", "name", "host", "db_name"})
	maxLifetimeClosedGaugeVec = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "gorm_dbstats_max_lifetime_closed",
		Help: "The total number of connections closed due to SetConnMaxLifetime.",
	}, []string{"type", "name", "host", "db_name"})
	maxIdleTimeClosedGaugeVec = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "gorm_dbstats_max_idletime_closed",
		Help: "The total number of connections closed due to SetConnMaxIdleTime.",
	}, []string{"type", "name", "host", "db_name"})
)

type Collector struct {
	instances map[string]*Client
	mux       sync.Mutex
}

func (c *Collector) Describe(descs chan<- *prometheus.Desc) {
	prometheus.DescribeByCollect(c, descs)
}

func (c *Collector) Collect(metrics chan<- prometheus.Metric) {
	c.mux.Lock()
	defer c.mux.Unlock()
	for _, instance := range c.instances {
		db, err := instance.database.DB()
		if err != nil {
			fmt.Println("failed to collect db metrics: failed to get db instance: ", err)
			continue
		}
		stats := db.Stats()
		connType := instance.options.GetType()
		dbName := instance.options.GetDBName()
		var host string
		peer, port := instance.options.GetPeer()
		if connType == "sqlite" {
			host = peer
		} else {
			host = fmt.Sprintf("%s:%d", peer, port)
		}
		maxOpenConnectionsGaugeVec.WithLabelValues(connType, instance.name, host, dbName).Set(float64(stats.MaxOpenConnections))
		openConnectionsGaugeVec.WithLabelValues(connType, instance.name, host, dbName).Set(float64(stats.OpenConnections))
		inUseGaugeVec.WithLabelValues(connType, instance.name, host, dbName).Set(float64(stats.InUse))
		idleGaugeVec.WithLabelValues(connType, instance.name, host, dbName).Set(float64(stats.Idle))
		waitCountGaugeVec.WithLabelValues(connType, instance.name, host, dbName).Set(float64(stats.WaitCount))
		waitDurationGaugeVec.WithLabelValues(connType, instance.name, host, dbName).Set(float64(stats.WaitDuration))
		maxIdleClosedGaugeVec.WithLabelValues(connType, instance.name, host, dbName).Set(float64(stats.MaxIdleClosed))
		maxLifetimeClosedGaugeVec.WithLabelValues(connType, instance.name, host, dbName).Set(float64(stats.MaxLifetimeClosed))
		maxIdleTimeClosedGaugeVec.WithLabelValues(connType, instance.name, host, dbName).Set(float64(stats.MaxIdleTimeClosed))

		metrics <- maxOpenConnectionsGaugeVec.WithLabelValues(connType, instance.name, host, dbName)
		metrics <- openConnectionsGaugeVec.WithLabelValues(connType, instance.name, host, dbName)
		metrics <- inUseGaugeVec.WithLabelValues(connType, instance.name, host, dbName)
		metrics <- idleGaugeVec.WithLabelValues(connType, instance.name, host, dbName)
		metrics <- waitCountGaugeVec.WithLabelValues(connType, instance.name, host, dbName)
		metrics <- waitDurationGaugeVec.WithLabelValues(connType, instance.name, host, dbName)
		metrics <- maxIdleClosedGaugeVec.WithLabelValues(connType, instance.name, host, dbName)
		metrics <- maxLifetimeClosedGaugeVec.WithLabelValues(connType, instance.name, host, dbName)
		metrics <- maxIdleTimeClosedGaugeVec.WithLabelValues(connType, instance.name, host, dbName)

	}
}

func (c *Collector) Register(db *Client) string {
	c.mux.Lock()
	defer c.mux.Unlock()
	id := uuid.Must(uuid.NewV4()).String()
	c.instances[id] = db
	return id
}
func (c *Collector) Unregister(id string) {
	c.mux.Lock()
	defer c.mux.Unlock()
	delete(c.instances, id)
}

var collector = &Collector{
	instances: make(map[string]*Client),
}

func init() {
	prometheus.MustRegister(collector)
}
