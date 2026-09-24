package initialize

import (
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"iptv-spider/global"
	"iptv-spider/initialize/internal"
	"iptv-spider/model"
	"os"
)

func Gorm() *gorm.DB {
	switch global.CONFIG.System.DbType {
	case "", "none":
		return nil
	case "mysql":
		return gormMysql()
	}
	panic("数据库连接出错！")
}

func MysqlTables(db *gorm.DB) {
	// 数据表初始化
	err := db.AutoMigrate(
		// 数据库模型
		model.Channel{},
		model.ChannelInfo{},
		model.AuthInfo{},
		model.EPGDetails{},
		model.M3u8Mapping{},
	)

	if err != nil {
		global.LOG.Error("register table failed", zap.Any("err", err))
		os.Exit(0)
	}
	global.LOG.Info("register table success")
}

func gormMysql() *gorm.DB {
	m := global.CONFIG.Mysql
	if m.Dbname == "" {
		panic("已启用 MySQL，请配置 mysql.db-name")
	}
	mysqlConfig := mysql.Config{
		DSN:                       m.Dsn(), // DSN data source name
		DefaultStringSize:         191,     // string 类型字段的默认长度
		DisableDatetimePrecision:  true,    // 禁用 datetime 精度，MySQL 5.6 之前的数据库不支持
		DontSupportRenameIndex:    true,    // 重命名索引时采用删除并新建的方式，MySQL 5.7 之前的数据库和 MariaDB 不支持重命名索引
		DontSupportRenameColumn:   true,    // 用 `change` 重命名列，MySQL 8 之前的数据库和 MariaDB 不支持重命名列
		SkipInitializeWithVersion: false,   // 根据版本自动配置
	}
	if db, err := gorm.Open(mysql.New(mysqlConfig), gormConfig()); err != nil {
		panic("MySQL 连接失败，请检查数据库地址、账号和权限")
	} else {
		sqlDB, _ := db.DB()
		sqlDB.SetMaxIdleConns(m.MaxIdleConns)
		sqlDB.SetMaxOpenConns(m.MaxOpenConns)
		return db
	}
}

func gormConfig() *gorm.Config {
	config := &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true}
	switch global.CONFIG.Mysql.LogMode {
	case "silent", "Silent":
		config.Logger = internal.Default.LogMode(logger.Silent)
	case "error", "Error":
		config.Logger = internal.Default.LogMode(logger.Error)
	case "warn", "Warn":
		config.Logger = internal.Default.LogMode(logger.Warn)
	case "info", "Info":
		config.Logger = internal.Default.LogMode(logger.Info)
	default:
		config.Logger = internal.Default.LogMode(logger.Info)
	}
	return config
}
