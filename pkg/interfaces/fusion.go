package interfaces

// FusionStrategy 数据融合策略
// 替代废弃的 fusion.FusionStrategy
type FusionStrategy string

const (
	FusionStrategyAverage        FusionStrategy = "average"
	FusionStrategyWeighted      FusionStrategy = "weighted"
	FusionStrategyKalman        FusionStrategy = "kalman"
	FusionStrategyBayesian      FusionStrategy = "bayesian"
	FusionStrategyDempsterShafer FusionStrategy = "dempster_shafer"
	FusionStrategyTimeSync      FusionStrategy = "time_sync"
	FusionStrategyInterpolation FusionStrategy = "interpolation"
	FusionStrategyExtrapolation FusionStrategy = "extrapolation"
	FusionStrategyMovingAverage FusionStrategy = "moving_average"
	FusionStrategyExponentialSMA FusionStrategy = "exponential_sma"
)

// DataFusion 数据融合接口
type DataFusion interface {
	// Fuse 融合多个数据样本
	Fuse(samples []DataSample, strategy FusionStrategy) (DataSample, error)

	// FuseWithWeights 使用权重融合数据
	FuseWithWeights(samples []DataSample, weights map[string]float64) (DataSample, error)
}
