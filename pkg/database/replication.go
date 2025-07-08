package database

import (
	"context"
	"fmt"
	"sync"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type DatabaseNode struct {
	Name     string    `json:"name"`
	Host     string    `json:"host"`
	Port     int       `json:"port"`
	Database string    `json:"database"`
	Username string    `json:"username"`
	Password string    `json:"password"`
	SSLMode  string    `json:"ssl_mode"`
	Role     string    `json:"role"`
	Weight   int       `json:"weight"`
	IsActive bool      `json:"is_active"`
	LastPing time.Time `json:"last_ping"`
}

type ReplicationConfig struct {
	MasterNode          DatabaseNode   `json:"master_node"`
	SlaveNodes          []DatabaseNode `json:"slave_nodes"`
	ReadReplicas        []DatabaseNode `json:"read_replicas"`
	MaxConnections      int            `json:"max_connections"`
	MaxIdleConns        int            `json:"max_idle_conns"`
	ConnMaxLifetime     time.Duration  `json:"conn_max_lifetime"`
	HealthCheckInterval time.Duration  `json:"health_check_interval"`
	FailoverEnabled     bool           `json:"failover_enabled"`
	AutoFailbackEnabled bool           `json:"auto_failback_enabled"`
}

type DatabaseCluster struct {
	config     ReplicationConfig
	masterDB   *gorm.DB
	slaveDBs   []*gorm.DB
	readDBs    []*gorm.DB
	mu         sync.RWMutex
	healthChan chan HealthCheckResult
	ctx        context.Context
	cancel     context.CancelFunc
}

type HealthCheckResult struct {
	Node    DatabaseNode  `json:"node"`
	Status  string        `json:"status"`
	Error   error         `json:"error,omitempty"`
	Latency time.Duration `json:"latency"`
}

func NewDatabaseCluster(config ReplicationConfig) (*DatabaseCluster, error) {
	ctx, cancel := context.WithCancel(context.Background())

	cluster := &DatabaseCluster{
		config:     config,
		healthChan: make(chan HealthCheckResult, 100),
		ctx:        ctx,
		cancel:     cancel,
	}

	masterDB, err := cluster.connectToNode(config.MasterNode)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to master: %w", err)
	}
	cluster.masterDB = masterDB

	for _, slaveNode := range config.SlaveNodes {
		slaveDB, err := cluster.connectToNode(slaveNode)
		if err != nil {
			fmt.Printf("Warning: failed to connect to slave %s: %v\n", slaveNode.Name, err)
			continue
		}
		cluster.slaveDBs = append(cluster.slaveDBs, slaveDB)
	}

	for _, readNode := range config.ReadReplicas {
		readDB, err := cluster.connectToNode(readNode)
		if err != nil {
			fmt.Printf("Warning: failed to connect to read replica %s: %v\n", readNode.Name, err)
			continue
		}
		cluster.readDBs = append(cluster.readDBs, readDB)
	}

	go cluster.startHealthMonitoring()

	return cluster, nil
}

func (c *DatabaseCluster) connectToNode(node DatabaseNode) (*gorm.DB, error) {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		node.Host, node.Port, node.Username, node.Password, node.Database, node.SSLMode)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	sqlDB.SetMaxOpenConns(c.config.MaxConnections)
	sqlDB.SetMaxIdleConns(c.config.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(c.config.ConnMaxLifetime)

	return db, nil
}

func (c *DatabaseCluster) GetMasterDB() *gorm.DB {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.masterDB
}

func (c *DatabaseCluster) GetSlaveDB() *gorm.DB {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.slaveDBs) == 0 {
		return c.masterDB
	}

	index := time.Now().UnixNano() % int64(len(c.slaveDBs))
	return c.slaveDBs[index]
}

func (c *DatabaseCluster) GetReadDB() *gorm.DB {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.readDBs) == 0 {
		return c.GetSlaveDB()
	}

	totalWeight := 0
	for _, node := range c.config.ReadReplicas {
		if node.IsActive {
			totalWeight += node.Weight
		}
	}

	if totalWeight == 0 {
		return c.GetSlaveDB()
	}

	index := time.Now().UnixNano() % int64(totalWeight)
	currentWeight := 0
	for i, node := range c.config.ReadReplicas {
		if node.IsActive {
			currentWeight += node.Weight
			if int64(currentWeight) > index {
				return c.readDBs[i]
			}
		}
	}

	return c.readDBs[0]
}

func (c *DatabaseCluster) startHealthMonitoring() {
	ticker := time.NewTicker(c.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.performHealthCheck()
		}
	}
}

func (c *DatabaseCluster) performHealthCheck() {
	go c.checkNodeHealth(c.config.MasterNode, c.masterDB, "master")

	for i, slaveNode := range c.config.SlaveNodes {
		if i < len(c.slaveDBs) {
			go c.checkNodeHealth(slaveNode, c.slaveDBs[i], "slave")
		}
	}

	for i, readNode := range c.config.ReadReplicas {
		if i < len(c.readDBs) {
			go c.checkNodeHealth(readNode, c.readDBs[i], "read_replica")
		}
	}
}

func (c *DatabaseCluster) checkNodeHealth(node DatabaseNode, db *gorm.DB, nodeType string) {
	start := time.Now()

	sqlDB, err := db.DB()
	if err != nil {
		c.healthChan <- HealthCheckResult{
			Node:   node,
			Status: "unhealthy",
			Error:  err,
		}
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = sqlDB.PingContext(ctx)
	latency := time.Since(start)

	result := HealthCheckResult{
		Node:    node,
		Latency: latency,
	}

	if err != nil {
		result.Status = "unhealthy"
		result.Error = err

		c.updateNodeStatus(node.Name, false)

		if c.config.FailoverEnabled && nodeType == "master" {
			c.triggerFailover()
		}
	} else {
		result.Status = "healthy"
		c.updateNodeStatus(node.Name, true)
	}

	select {
	case c.healthChan <- result:
	default:
	}
}

func (c *DatabaseCluster) updateNodeStatus(nodeName string, isActive bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.config.MasterNode.Name == nodeName {
		c.config.MasterNode.IsActive = isActive
		c.config.MasterNode.LastPing = time.Now()
	}

	for i := range c.config.SlaveNodes {
		if c.config.SlaveNodes[i].Name == nodeName {
			c.config.SlaveNodes[i].IsActive = isActive
			c.config.SlaveNodes[i].LastPing = time.Now()
			break
		}
	}

	for i := range c.config.ReadReplicas {
		if c.config.ReadReplicas[i].Name == nodeName {
			c.config.ReadReplicas[i].IsActive = isActive
			c.config.ReadReplicas[i].LastPing = time.Now()
			break
		}
	}
}

func (c *DatabaseCluster) triggerFailover() {
	c.mu.Lock()
	defer c.mu.Unlock()

	var bestSlave *DatabaseNode
	for i := range c.config.SlaveNodes {
		if c.config.SlaveNodes[i].IsActive {
			if bestSlave == nil || c.config.SlaveNodes[i].Weight > bestSlave.Weight {
				bestSlave = &c.config.SlaveNodes[i]
			}
		}
	}

	if bestSlave != nil {
		oldMaster := c.config.MasterNode
		c.config.MasterNode = *bestSlave
		c.config.MasterNode.Role = "master"

		if newMasterDB, err := c.connectToNode(c.config.MasterNode); err == nil {
			c.masterDB = newMasterDB
		}

		oldMaster.Role = "slave"
		c.config.SlaveNodes = append(c.config.SlaveNodes, oldMaster)

		fmt.Printf("Failover completed: %s promoted to master\n", bestSlave.Name)
	}
}

func (c *DatabaseCluster) GetHealthStatus() map[string]HealthCheckResult {
	c.mu.RLock()
	defer c.mu.RUnlock()

	status := make(map[string]HealthCheckResult)

	for {
		select {
		case result := <-c.healthChan:
			status[result.Node.Name] = result
		default:
			goto done
		}
	}
done:

	return status
}

func (c *DatabaseCluster) Close() error {
	c.cancel()

	if c.masterDB != nil {
		if sqlDB, err := c.masterDB.DB(); err == nil {
			sqlDB.Close()
		}
	}

	for _, slaveDB := range c.slaveDBs {
		if sqlDB, err := slaveDB.DB(); err == nil {
			sqlDB.Close()
		}
	}

	for _, readDB := range c.readDBs {
		if sqlDB, err := readDB.DB(); err == nil {
			sqlDB.Close()
		}
	}

	return nil
}

func (c *DatabaseCluster) GetClusterStats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := map[string]interface{}{
		"master_active":        c.config.MasterNode.IsActive,
		"slave_count":          len(c.config.SlaveNodes),
		"active_slaves":        0,
		"read_replica_count":   len(c.config.ReadReplicas),
		"active_read_replicas": 0,
		"total_connections":    c.config.MaxConnections,
		"failover_enabled":     c.config.FailoverEnabled,
	}

	for _, slave := range c.config.SlaveNodes {
		if slave.IsActive {
			stats["active_slaves"] = stats["active_slaves"].(int) + 1
		}
	}

	// Count active read replicas
	for _, replica := range c.config.ReadReplicas {
		if replica.IsActive {
			stats["active_read_replicas"] = stats["active_read_replicas"].(int) + 1
		}
	}

	return stats
}
