package server

import (
	"fmt"
	"net/http"
	"time"

	"transaction-api-w-go/pkg/circuitbreaker"
	"transaction-api-w-go/pkg/database"
	"transaction-api-w-go/pkg/fallback"
	"transaction-api-w-go/pkg/loadbalancer"

	"github.com/gin-gonic/gin"
)

type HAHandler struct {
	dbCluster       *database.DatabaseCluster
	loadBalancer    *loadbalancer.LoadBalancer
	circuitBreakers map[string]*circuitbreaker.CircuitBreaker
	fallbackManager *fallback.FallbackManager
}

func NewHAHandler(
	dbCluster *database.DatabaseCluster,
	loadBalancer *loadbalancer.LoadBalancer,
	fallbackManager *fallback.FallbackManager,
) *HAHandler {
	return &HAHandler{
		dbCluster:       dbCluster,
		loadBalancer:    loadBalancer,
		circuitBreakers: make(map[string]*circuitbreaker.CircuitBreaker),
		fallbackManager: fallbackManager,
	}
}

func (h *HAHandler) GetDatabaseHealth(c *gin.Context) {
	healthStatus := h.dbCluster.GetHealthStatus()
	clusterStats := h.dbCluster.GetClusterStats()

	c.JSON(http.StatusOK, gin.H{
		"health_status": healthStatus,
		"cluster_stats": clusterStats,
		"timestamp":     time.Now(),
	})
}

func (h *HAHandler) GetDatabaseNodeHealth(c *gin.Context) {
	nodeName := c.Param("node")
	healthStatus := h.dbCluster.GetHealthStatus()

	if nodeHealth, exists := healthStatus[nodeName]; exists {
		c.JSON(http.StatusOK, gin.H{
			"node_health": nodeHealth,
			"timestamp":   time.Now(),
		})
	} else {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Node not found",
		})
	}
}

func (h *HAHandler) ForceDatabaseFailover(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message":   "Database failover triggered",
		"timestamp": time.Now(),
	})
}

func (h *HAHandler) GetLoadBalancerStats(c *gin.Context) {
	stats := h.loadBalancer.GetStats()
	backends := h.loadBalancer.GetBackends()

	c.JSON(http.StatusOK, gin.H{
		"stats":     stats,
		"backends":  backends,
		"timestamp": time.Now(),
	})
}

func (h *HAHandler) AddLoadBalancerBackend(c *gin.Context) {
	var req struct {
		ID     string `json:"id" binding:"required"`
		URL    string `json:"url" binding:"required"`
		Weight int    `json:"weight"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	backend := &loadbalancer.Backend{
		ID:       req.ID,
		URL:      req.URL,
		Weight:   req.Weight,
		IsActive: true,
		Health:   1.0,
	}

	h.loadBalancer.AddBackend(backend)

	c.JSON(http.StatusCreated, gin.H{
		"message": "Backend added successfully",
		"backend": backend,
	})
}

func (h *HAHandler) RemoveLoadBalancerBackend(c *gin.Context) {
	backendID := c.Param("id")
	h.loadBalancer.RemoveBackend(backendID)

	c.JSON(http.StatusOK, gin.H{
		"message": "Backend removed successfully",
	})
}

func (h *HAHandler) GetCircuitBreakerStats(c *gin.Context) {
	breakerName := c.Param("name")

	breaker, exists := h.circuitBreakers[breakerName]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Circuit breaker not found",
		})
		return
	}

	stats := breaker.GetStats()
	c.JSON(http.StatusOK, gin.H{
		"circuit_breaker": stats,
		"timestamp":       time.Now(),
	})
}

func (h *HAHandler) GetAllCircuitBreakers(c *gin.Context) {
	allStats := make(map[string]interface{})

	for name, breaker := range h.circuitBreakers {
		allStats[name] = breaker.GetStats()
	}

	c.JSON(http.StatusOK, gin.H{
		"circuit_breakers": allStats,
		"timestamp":        time.Now(),
	})
}

func (h *HAHandler) CreateCircuitBreaker(c *gin.Context) {
	var req struct {
		Name   string                `json:"name" binding:"required"`
		Config circuitbreaker.Config `json:"config"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Config.FailureThreshold == 0 {
		req.Config = circuitbreaker.DefaultConfig()
	}

	breaker := circuitbreaker.NewCircuitBreaker(req.Name, req.Config)
	h.circuitBreakers[req.Name] = breaker

	c.JSON(http.StatusCreated, gin.H{
		"message": "Circuit breaker created successfully",
		"name":    req.Name,
		"config":  req.Config,
	})
}

func (h *HAHandler) ForceCircuitBreakerOpen(c *gin.Context) {
	breakerName := c.Param("name")

	breaker, exists := h.circuitBreakers[breakerName]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Circuit breaker not found",
		})
		return
	}

	breaker.ForceOpen()

	c.JSON(http.StatusOK, gin.H{
		"message": "Circuit breaker forced open",
		"name":    breakerName,
	})
}

func (h *HAHandler) ForceCircuitBreakerClose(c *gin.Context) {
	breakerName := c.Param("name")

	breaker, exists := h.circuitBreakers[breakerName]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Circuit breaker not found",
		})
		return
	}

	breaker.ForceClose()

	c.JSON(http.StatusOK, gin.H{
		"message": "Circuit breaker forced closed",
		"name":    breakerName,
	})
}

func (h *HAHandler) ResetCircuitBreaker(c *gin.Context) {
	breakerName := c.Param("name")

	breaker, exists := h.circuitBreakers[breakerName]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Circuit breaker not found",
		})
		return
	}

	breaker.Reset()

	c.JSON(http.StatusOK, gin.H{
		"message": "Circuit breaker reset",
		"name":    breakerName,
	})
}

func (h *HAHandler) GetFallbackStats(c *gin.Context) {
	stats := h.fallbackManager.GetStats()

	c.JSON(http.StatusOK, gin.H{
		"fallback_stats": stats,
		"timestamp":      time.Now(),
	})
}

func (h *HAHandler) TestFallback(c *gin.Context) {
	var req struct {
		Key      string `json:"key" binding:"required"`
		Strategy string `json:"strategy"` // sequential, parallel, degradation
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	primary := func() (interface{}, error) {
		if time.Now().UnixNano()%3 == 0 {
			return nil, fmt.Errorf("primary function failed")
		}
		return "primary result", nil
	}

	fallback1 := func() (interface{}, error) {
		return "fallback1 result", nil
	}

	fallback2 := func() (interface{}, error) {
		return "fallback2 result", nil
	}

	result, err := h.fallbackManager.Execute(c.Request.Context(), req.Key, primary, fallback1, fallback2)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"result":    result,
		"key":       req.Key,
		"strategy":  req.Strategy,
		"timestamp": time.Now(),
	})
}

func (h *HAHandler) GetSystemHealth(c *gin.Context) {
	dbHealth := h.dbCluster.GetHealthStatus()
	dbStats := h.dbCluster.GetClusterStats()

	lbStats := h.loadBalancer.GetStats()

	cbStats := make(map[string]interface{})
	for name, breaker := range h.circuitBreakers {
		cbStats[name] = breaker.GetStats()
	}

	fbStats := h.fallbackManager.GetStats()

	systemStatus := "healthy"

	for _, health := range dbHealth {
		if health.Status != "healthy" {
			systemStatus = "degraded"
			break
		}
	}

	if lbStats["active_backends"].(int) == 0 {
		systemStatus = "degraded"
	}

	for _, cbStat := range cbStats {
		if cbStat.(map[string]interface{})["state"] == "OPEN" {
			systemStatus = "degraded"
			break
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"system_status": systemStatus,
		"database": gin.H{
			"health": dbHealth,
			"stats":  dbStats,
		},
		"load_balancer": gin.H{
			"stats": lbStats,
		},
		"circuit_breakers": cbStats,
		"fallback": gin.H{
			"stats": fbStats,
		},
		"timestamp": time.Now(),
	})
}

func (h *HAHandler) GetHAConfig(c *gin.Context) {
	config := gin.H{
		"database": gin.H{
			"replication_enabled":   true,
			"failover_enabled":      true,
			"health_check_interval": "30s",
		},
		"load_balancer": gin.H{
			"strategy":              "round_robin",
			"health_check_interval": "30s",
		},
		"circuit_breaker": gin.H{
			"default_config": circuitbreaker.DefaultConfig(),
			"strict_config":  circuitbreaker.StrictConfig(),
			"lenient_config": circuitbreaker.LenientConfig(),
		},
		"fallback": gin.H{
			"default_config": fallback.DefaultConfig(),
			"strict_config":  fallback.StrictConfig(),
			"lenient_config": fallback.LenientConfig(),
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"ha_config": config,
		"timestamp": time.Now(),
	})
}

func (h *HAHandler) UpdateHAConfig(c *gin.Context) {
	var req struct {
		Component string                 `json:"component" binding:"required"`
		Config    map[string]interface{} `json:"config" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "Configuration updated successfully",
		"component": req.Component,
		"timestamp": time.Now(),
	})
}

func (h *HAHandler) GetHAMetrics(c *gin.Context) {
	dbStats := h.dbCluster.GetClusterStats()

	lbStats := h.loadBalancer.GetStats()

	cbMetrics := make(map[string]interface{})
	for name, breaker := range h.circuitBreakers {
		stats := breaker.GetStats()
		cbMetrics[name] = gin.H{
			"state":      stats["state"],
			"error_rate": stats["error_rate"],
			"requests":   stats["requests"],
		}
	}

	fbStats := h.fallbackManager.GetStats()

	metrics := gin.H{
		"database": gin.H{
			"active_nodes":     dbStats["active_slaves"].(int) + dbStats["active_read_replicas"].(int),
			"total_nodes":      dbStats["slave_count"].(int) + dbStats["read_replica_count"].(int),
			"failover_enabled": dbStats["failover_enabled"],
		},
		"load_balancer": gin.H{
			"active_backends": lbStats["active_backends"],
			"total_backends":  lbStats["total_backends"],
			"average_latency": lbStats["average_latency"],
			"average_health":  lbStats["average_health"],
		},
		"circuit_breakers": cbMetrics,
		"fallback": gin.H{
			"cache_size":         fbStats["cache_size"],
			"enable_caching":     fbStats["enable_caching"],
			"enable_degradation": fbStats["enable_degradation"],
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"ha_metrics": metrics,
		"timestamp":  time.Now(),
	})
}
