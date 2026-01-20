package anomaly

import (
	"math"
)

// SimilarityMetric 相似度度量方法
type SimilarityMetric int

const (
	MetricEuclidean     SimilarityMetric = iota // 欧氏距离
	MetricCosine                                 // 余弦相似度
	MetricPearson                                // 皮尔逊相关系数
	MetricDTW                                    // 动态时间规整
	MetricManhattan                              // 曼哈顿距离
	MetricChebyshev                              // 切比雪夫距离
)

// SimilarityCalculator 相似度计算器
type SimilarityCalculator struct {
	metric SimilarityMetric
}

// NewSimilarityCalculator 创建相似度计算器
func NewSimilarityCalculator(metric SimilarityMetric) *SimilarityCalculator {
	return &SimilarityCalculator{
		metric: metric,
	}
}

// Calculate 计算两个向量的相似度
func (sc *SimilarityCalculator) Calculate(a, b []float64) float64 {
	switch sc.metric {
	case MetricEuclidean:
		return euclideanSimilarity(a, b)
	case MetricCosine:
		return cosineSimilarity(a, b)
	case MetricPearson:
		return pearsonCorrelation(a, b)
	case MetricDTW:
		return dtwSimilarity(a, b)
	case MetricManhattan:
		return manhattanSimilarity(a, b)
	case MetricChebyshev:
		return chebyshevSimilarity(a, b)
	default:
		return cosineSimilarity(a, b)
	}
}

// euclideanDistance 欧氏距离
func euclideanDistance(a, b []float64) float64 {
	if len(a) != len(b) {
		return math.MaxFloat64
	}

	sum := 0.0
	for i := range a {
		diff := a[i] - b[i]
		sum += diff * diff
	}
	return math.Sqrt(sum)
}

// euclideanSimilarity 欧氏相似度（0-1，1表示完全相同）
func euclideanSimilarity(a, b []float64) float64 {
	distance := euclideanDistance(a, b)
	// 转换为相似度：1 / (1 + distance)
	return 1.0 / (1.0 + distance)
}

// cosineSimilarity 余弦相似度（-1到1，1表示完全相同）
func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0.0
	}

	dotProduct := 0.0
	normA := 0.0
	normB := 0.0

	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0.0
	}

	similarity := dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))

	// 归一化到 0-1
	return (similarity + 1.0) / 2.0
}

// pearsonCorrelation 皮尔逊相关系数（-1到1，1表示完全正相关）
func pearsonCorrelation(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0.0
	}

	n := float64(len(a))

	// 计算均值
	meanA := 0.0
	meanB := 0.0
	for i := range a {
		meanA += a[i]
		meanB += b[i]
	}
	meanA /= n
	meanB /= n

	// 计算协方差和标准差
	covariance := 0.0
	varA := 0.0
	varB := 0.0

	for i := range a {
		diffA := a[i] - meanA
		diffB := b[i] - meanB
		covariance += diffA * diffB
		varA += diffA * diffA
		varB += diffB * diffB
	}

	if varA == 0 || varB == 0 {
		return 0.0
	}

	correlation := covariance / math.Sqrt(varA*varB)

	// 归一化到 0-1
	return (correlation + 1.0) / 2.0
}

// dtwDistance 动态时间规整距离
func dtwDistance(a, b []float64) float64 {
	n := len(a)
	m := len(b)

	if n == 0 || m == 0 {
		return math.MaxFloat64
	}

	// 创建DTW矩阵
	dtw := make([][]float64, n+1)
	for i := range dtw {
		dtw[i] = make([]float64, m+1)
		for j := range dtw[i] {
			dtw[i][j] = math.MaxFloat64
		}
	}
	dtw[0][0] = 0

	// 填充DTW矩阵
	for i := 1; i <= n; i++ {
		for j := 1; j <= m; j++ {
			cost := math.Abs(a[i-1] - b[j-1])
			dtw[i][j] = cost + math.Min(
				math.Min(dtw[i-1][j], dtw[i][j-1]),
				dtw[i-1][j-1],
			)
		}
	}

	return dtw[n][m]
}

// dtwSimilarity DTW相似度（0-1，1表示完全相同）
func dtwSimilarity(a, b []float64) float64 {
	distance := dtwDistance(a, b)
	// 转换为相似度
	return 1.0 / (1.0 + distance)
}

// manhattanDistance 曼哈顿距离
func manhattanDistance(a, b []float64) float64 {
	if len(a) != len(b) {
		return math.MaxFloat64
	}

	sum := 0.0
	for i := range a {
		sum += math.Abs(a[i] - b[i])
	}
	return sum
}

// manhattanSimilarity 曼哈顿相似度
func manhattanSimilarity(a, b []float64) float64 {
	distance := manhattanDistance(a, b)
	return 1.0 / (1.0 + distance)
}

// chebyshevDistance 切比雪夫距离
func chebyshevDistance(a, b []float64) float64 {
	if len(a) != len(b) {
		return math.MaxFloat64
	}

	maxDiff := 0.0
	for i := range a {
		diff := math.Abs(a[i] - b[i])
		if diff > maxDiff {
			maxDiff = diff
		}
	}
	return maxDiff
}

// chebyshevSimilarity 切比雪夫相似度
func chebyshevSimilarity(a, b []float64) float64 {
	distance := chebyshevDistance(a, b)
	return 1.0 / (1.0 + distance)
}

// PatternMatcher 模式匹配器
type PatternMatcher struct {
	calculator *SimilarityCalculator
	threshold  float64
}

// NewPatternMatcher 创建模式匹配器
func NewPatternMatcher(metric SimilarityMetric, threshold float64) *PatternMatcher {
	return &PatternMatcher{
		calculator: NewSimilarityCalculator(metric),
		threshold:  threshold,
	}
}

// Match 匹配模式
func (pm *PatternMatcher) Match(pattern, data []float64) (bool, float64) {
	similarity := pm.calculator.Calculate(pattern, data)
	return similarity >= pm.threshold, similarity
}

// FindBestMatch 在多个模式中找到最佳匹配
func (pm *PatternMatcher) FindBestMatch(patterns [][]float64, data []float64) (int, float64) {
	bestIndex := -1
	bestSimilarity := 0.0

	for i, pattern := range patterns {
		similarity := pm.calculator.Calculate(pattern, data)
		if similarity > bestSimilarity {
			bestSimilarity = similarity
			bestIndex = i
		}
	}

	return bestIndex, bestSimilarity
}

// SlidingWindowMatcher 滑动窗口匹配器
type SlidingWindowMatcher struct {
	calculator  *SimilarityCalculator
	threshold   float64
	windowSize  int
}

// NewSlidingWindowMatcher 创建滑动窗口匹配器
func NewSlidingWindowMatcher(metric SimilarityMetric, threshold float64, windowSize int) *SlidingWindowMatcher {
	return &SlidingWindowMatcher{
		calculator:  NewSimilarityCalculator(metric),
		threshold:   threshold,
		windowSize:  windowSize,
	}
}

// MatchResult 匹配结果
type MatchResult struct {
	Position   int
	Similarity float64
	Matched    bool
}

// FindMatches 在时间序列中查找匹配的模式
func (swm *SlidingWindowMatcher) FindMatches(pattern, timeseries []float64) []MatchResult {
	if len(timeseries) < len(pattern) {
		return nil
	}

	results := make([]MatchResult, 0)

	for i := 0; i <= len(timeseries)-len(pattern); i++ {
		window := timeseries[i : i+len(pattern)]
		similarity := swm.calculator.Calculate(pattern, window)

		result := MatchResult{
			Position:   i,
			Similarity: similarity,
			Matched:    similarity >= swm.threshold,
		}

		if result.Matched {
			results = append(results, result)
		}
	}

	return results
}

// NormalizeTimeSeries 归一化时间序列
func NormalizeTimeSeries(data []float64) []float64 {
	if len(data) == 0 {
		return data
	}

	// 计算均值和标准差
	mean := 0.0
	for _, v := range data {
		mean += v
	}
	mean /= float64(len(data))

	variance := 0.0
	for _, v := range data {
		diff := v - mean
		variance += diff * diff
	}
	stddev := math.Sqrt(variance / float64(len(data)))

	if stddev == 0 {
		// 所有值相同，返回零向量
		normalized := make([]float64, len(data))
		return normalized
	}

	// Z-score 归一化
	normalized := make([]float64, len(data))
	for i, v := range data {
		normalized[i] = (v - mean) / stddev
	}

	return normalized
}

// ExtractFeatures 从时间序列中提取特征
func ExtractFeatures(data []float64) []float64 {
	if len(data) == 0 {
		return nil
	}

	features := make([]float64, 0, 8)

	// 统计特征
	mean := 0.0
	for _, v := range data {
		mean += v
	}
	mean /= float64(len(data))
	features = append(features, mean)

	// 标准差
	variance := 0.0
	for _, v := range data {
		diff := v - mean
		variance += diff * diff
	}
	stddev := math.Sqrt(variance / float64(len(data)))
	features = append(features, stddev)

	// 最小值和最大值
	min := data[0]
	max := data[0]
	for _, v := range data {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	features = append(features, min)
	features = append(features, max)
	features = append(features, max-min) // 范围

	// 峰度和偏度（简化版本）
	skewness := 0.0
	kurtosis := 0.0
	for _, v := range data {
		diff := v - mean
		skewness += diff * diff * diff
		kurtosis += diff * diff * diff * diff
	}
	if stddev != 0 {
		skewness = skewness / (float64(len(data)) * math.Pow(stddev, 3))
		kurtosis = kurtosis/(float64(len(data))*math.Pow(stddev, 4)) - 3
	}
	features = append(features, skewness)
	features = append(features, kurtosis)

	// 趋势（最后值 - 第一值）
	trend := data[len(data)-1] - data[0]
	features = append(features, trend)

	return features
}
