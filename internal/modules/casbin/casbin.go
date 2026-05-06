package casbinauth

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/casbin/casbin/v2"
	fileadapter "github.com/casbin/casbin/v2/persist/file-adapter"
	"github.com/ouqiang/gocron/internal/modules/logger"
)

const (
	RoleAdmin    = "admin"
	RoleOperator = "operator"
	RoleViewer   = "viewer"

	// GlobalDomain 全局租户域，适用于超级管理员
	GlobalDomain = "*"
)

var (
	once     sync.Once
	enforcer *casbin.Enforcer
)

// Init 初始化 casbin 执行器，confDir 为配置文件目录
// 若 rbac_model.conf 不存在则静默跳过，casbin 权限检查将被禁用
func Init(confDir string) {
	once.Do(func() {
		modelFile := filepath.Join(confDir, "rbac_model.conf")
		policyFile := filepath.Join(confDir, "policy.csv")

		// 如果模型文件不存在，跳过初始化（首次安装前无需 casbin）
		if _, err := os.Stat(modelFile); os.IsNotExist(err) {
			logger.Info("casbin 模型文件不存在，跳过权限管理初始化")
			return
		}
		// 如果策略文件不存在，创建空文件
		if _, err := os.Stat(policyFile); os.IsNotExist(err) {
			f, err := os.Create(policyFile)
			if err != nil {
				logger.Warnf("创建 casbin 策略文件失败: %s", err)
				return
			}
			f.Close()
		}

		adapter := fileadapter.NewAdapter(policyFile)
		e, err := casbin.NewEnforcer(modelFile, adapter)
		if err != nil {
			logger.Warnf("casbin 初始化失败: %s", err)
			return
		}
		e.EnableLog(false)
		enforcer = e
	})
}

// Enforcer 返回全局 casbin 执行器
func Enforcer() *casbin.Enforcer {
	return enforcer
}

// HasPermission 检查 subject 在指定域 domain 下对 obj 是否具有 act 权限
// isAdmin 为 true 时跳过检查，直接允许
func HasPermission(subject, domain, obj, act string) bool {
	if enforcer == nil {
		return true
	}
	ok, err := enforcer.Enforce(subject, domain, obj, act)
	if err != nil {
		logger.Warnf("casbin 鉴权异常: %s", err)
		return false
	}
	return ok
}

// AddRoleForUser 为用户分配角色（在指定域内）
func AddRoleForUser(user, role, domain string) error {
	_, err := enforcer.AddRoleForUserInDomain(user, role, domain)
	return err
}

// RemoveRoleForUser 删除用户在指定域内的角色
func RemoveRoleForUser(user, role, domain string) error {
	_, err := enforcer.DeleteRoleForUserInDomain(user, role, domain)
	return err
}

// GetRolesForUser 获取用户在指定域内的所有角色
func GetRolesForUser(user, domain string) []string {
	return enforcer.GetRolesForUserInDomain(user, domain)
}

// GetUsersForRole 获取指定域内某角色的所有用户
func GetUsersForRole(role, domain string) []string {
	return enforcer.GetUsersForRoleInDomain(role, domain)
}

// GetAllPolicies 获取所有策略规则
func GetAllPolicies() [][]string {
	policies, _ := enforcer.GetPolicy()
	return policies
}

// GetAllGroupingPolicies 获取所有角色分配规则
func GetAllGroupingPolicies() [][]string {
	groupings, _ := enforcer.GetGroupingPolicy()
	return groupings
}

// AddPolicy 添加一条策略规则
func AddPolicy(sub, dom, obj, act string) error {
	_, err := enforcer.AddPolicy(sub, dom, obj, act)
	return err
}

// RemovePolicy 删除一条策略规则
func RemovePolicy(sub, dom, obj, act string) error {
	_, err := enforcer.RemovePolicy(sub, dom, obj, act)
	return err
}

// SavePolicy 持久化当前策略到文件
func SavePolicy() error {
	return enforcer.SavePolicy()
}
