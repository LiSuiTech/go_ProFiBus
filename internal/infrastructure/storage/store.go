package storage

import (
	root "go_ProFiBus/storage"
)

// PostgresStore 为根 storage 包中 PostgresStore 的类型别名，便于本包内各 Repository 使用同一类型。
type PostgresStore = root.PostgresStore
