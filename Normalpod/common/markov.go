package common

import (
	"math"
	"math/rand"
	"time"
)

// GaussMarkov3D 3D高斯马尔科夫移动模型
type GaussMarkov3D struct {
	// 模型参数
	Alpha                   float64       // 随机性控制参数 (0.0-1.0)
	PositionUpdateInterval  time.Duration // 位置更新间隔
	DirectionUpdateInterval time.Duration // 方向更新间隔

	// 状态变量
	Position      [3]float64 // 当前位置 [x, y, z]
	Velocity      [3]float64 // 当前速度 [vx, vy, vz]
	Direction     float64    // 水平方向 (弧度)
	Pitch         float64    // 俯仰角 (弧度)
	VelocityMean  float64    // 平均速度
	DirectionMean float64    // 平均方向
	PitchMean     float64    // 平均俯仰角

	// 边界设置
	MinX, MaxX float64
	MinY, MaxY float64
	MinZ, MaxZ float64
	B1, B2, B3 float64 // 边界缓冲区

	// 内部状态
	lastDirectionUpdate time.Time
	rng                 *rand.Rand
	moveCounter         int
}

// NewGaussMarkov3D 创建新的3D高斯马尔科夫模型
func NewGaussMarkov3D(seed int64, config *GaussMarkovConfig) *GaussMarkov3D {
	gm := &GaussMarkov3D{
		Alpha:                   config.Alpha,
		PositionUpdateInterval:  config.PositionUpdateInterval,
		DirectionUpdateInterval: config.DirectionUpdateInterval,
		Position:                config.InitialPosition,
		Velocity:                config.InitialVelocity,
		Direction:               config.InitialDirection,
		Pitch:                   config.InitialPitch,
		VelocityMean:            config.VelocityMean,
		DirectionMean:           config.DirectionMean,
		PitchMean:               config.PitchMean,
		MinX:                    config.MinX,
		MaxX:                    config.MaxX,
		MinY:                    config.MinY,
		MaxY:                    config.MaxY,
		MinZ:                    config.MinZ,
		MaxZ:                    config.MaxZ,
		B1:                      config.B1,
		B2:                      config.B2,
		B3:                      config.B3,
		lastDirectionUpdate:     time.Now(),
		moveCounter:             1,
	}

	// 初始化随机数生成器
	gm.rng = rand.New(rand.NewSource(seed))

	return gm
}

// GaussMarkovConfig 模型配置参数
type GaussMarkovConfig struct {
	Alpha                   float64
	PositionUpdateInterval  time.Duration
	DirectionUpdateInterval time.Duration
	InitialPosition         [3]float64
	InitialVelocity         [3]float64
	InitialDirection        float64
	InitialPitch            float64
	VelocityMean            float64
	DirectionMean           float64
	PitchMean               float64
	MinX, MaxX              float64
	MinY, MaxY              float64
	MinZ, MaxZ              float64
	B1, B2, B3              float64
}

// Update 更新模型状态并返回新位置
func (gm *GaussMarkov3D) Update() [3]float64 {
	now := time.Now()
	timeDelta := now.Sub(gm.lastDirectionUpdate).Seconds()

	// 当前位置和速度
	curPosition := gm.Position
	curVelocity := gm.Velocity
	curDirection := gm.Direction
	curPitch := gm.Pitch
	nextVelocity := [3]float64{}
	nextDirection := curDirection
	nextPitch := curPitch
	// 1. 更新位置
	nextPosition := [3]float64{
		curPosition[0] + curVelocity[0]*timeDelta,
		curPosition[1] + curVelocity[1]*timeDelta,
		curPosition[2] + curVelocity[2]*timeDelta,
	}

	// 计算当前速度
	nextSpeed := math.Sqrt(
		curVelocity[0]*curVelocity[0] +
			curVelocity[1]*curVelocity[1] +
			curVelocity[2]*curVelocity[2])

	// 3. 定期更新方向和速度
	if time.Since(gm.lastDirectionUpdate) >= gm.DirectionUpdateInterval {
		alpha2 := 1.0 - gm.Alpha
		alpha3 := math.Sqrt(1.0 - gm.Alpha*gm.Alpha)
		// 计算新速度（即使curSpeed=0也更新）
		nextSpeed := gm.Alpha*nextSpeed +
			alpha2*gm.VelocityMean +
			alpha3*0.5*gm.rng.NormFloat64() // 减小随机波动

		// 计算新方向（添加小范围随机扰动）
		nextDirection = gm.Alpha*curDirection +
			alpha2*gm.DirectionMean +
			alpha3*0.1*gm.rng.NormFloat64()

		// 计算新俯仰角
		nextPitch = gm.Alpha*curPitch +
			alpha2*gm.PitchMean +
			alpha3*0.05*gm.rng.NormFloat64()

		// 更新速度分量
		nextVelocity[0] = nextSpeed * math.Cos(nextDirection) * math.Cos(nextPitch)
		nextVelocity[1] = nextSpeed * math.Sin(nextDirection) * math.Cos(nextPitch)
		nextVelocity[2] = nextSpeed * math.Sin(nextPitch)

		gm.lastDirectionUpdate = now
	}

	// 2. 边界检测和反弹
	nextPosition, nextVelocity, nextDirection, nextPitch = gm.boundaryTest(
		nextPosition, nextVelocity, nextDirection, curPitch)

	// 更新模型状态
	gm.Position = nextPosition
	gm.Velocity = nextVelocity
	gm.Direction = nextDirection
	gm.Pitch = nextPitch

	return nextPosition
}

// 修复后的boundaryTest函数
func (gm *GaussMarkov3D) boundaryTest(
	position [3]float64,
	velocity [3]float64,
	direction float64,
	pitch float64,
) ([3]float64, [3]float64, float64, float64) {

	// 1. 边界检测和反弹
	if position[0] < gm.MinX+gm.B1 {
		position[0] = gm.MinX + gm.B1
		velocity[0] = math.Abs(velocity[0]) // 确保正方向
	} else if position[0] > gm.MaxX-gm.B1 {
		position[0] = gm.MaxX - gm.B1
		velocity[0] = -math.Abs(velocity[0]) // 确保负方向
	}

	if position[1] < gm.MinY+gm.B2 {
		position[1] = gm.MinY + gm.B2
		velocity[1] = math.Abs(velocity[1])
	} else if position[1] > gm.MaxY-gm.B2 {
		position[1] = gm.MaxY - gm.B2
		velocity[1] = -math.Abs(velocity[1])
	}

	if position[2] < gm.MinZ+gm.B3 {
		position[2] = gm.MinZ + gm.B3
		velocity[2] = math.Abs(velocity[2])
	} else if position[2] > gm.MaxZ-gm.B3 {
		position[2] = gm.MaxZ - gm.B3
		velocity[2] = -math.Abs(velocity[2])
	}

	// 2. 根据新速度重新计算方向
	vx, vy, vz := velocity[0], velocity[1], velocity[2]
	horizontalSpeed := math.Sqrt(vx*vx + vy*vy)

	newDirection := direction
	newPitch := pitch

	if horizontalSpeed > 0 {
		newDirection = math.Atan2(vy, vx)
		newPitch = math.Atan2(vz, horizontalSpeed)
	}

	// 3. 更新平均值
	gm.DirectionMean = newDirection
	gm.PitchMean = newPitch

	return position, velocity, newDirection, newPitch
}

// GetPosition 获取当前位置
func (gm *GaussMarkov3D) GetPosition() [3]float64 {
	return gm.Position
}

func (gm *GaussMarkov3D) SetBundary(minX, maxX, minY, maxY, minZ, maxZ float64) {
	gm.MinX = minX
	gm.MaxX = maxX
	gm.MinY = minY
	gm.MaxY = maxY
	gm.MinZ = minZ
	gm.MaxZ = maxZ
}

func Markov(speed float64) *GaussMarkov3D {
	// 使用 math.Pi 替代 np.pi
	direction := rand.Float64() * 2 * math.Pi // [0, 2π)
	pitch := rand.Float64()*0.1 - 0.05        // [-0.05, 0.05)

	velocity := [3]float64{
		speed * math.Cos(direction) * math.Cos(pitch),
		speed * math.Sin(direction) * math.Cos(pitch),
		speed * math.Sin(pitch),
	}

	config := &GaussMarkovConfig{
		Alpha:                   0.85,
		PositionUpdateInterval:  1000 * time.Millisecond,
		DirectionUpdateInterval: 5000 * time.Millisecond,
		InitialPosition:         [3]float64{0, 0, 0},
		InitialVelocity:         velocity,
		InitialDirection:        direction,
		InitialPitch:            pitch,
		VelocityMean:            speed,
		DirectionMean:           direction,
		PitchMean:               pitch,
		MinX:                    0,
		MaxX:                    100,
		MinY:                    0,
		MaxY:                    100,
		MinZ:                    0,
		MaxZ:                    50,
		B1:                      1,
		B2:                      1,
		B3:                      1,
	}

	gm := NewGaussMarkov3D(time.Now().UnixNano(), config)

	// 模拟运行代码应移到函数外部（调用者控制）
	return gm
}
