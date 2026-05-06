package admin

import (
	casbinauth "github.com/ouqiang/gocron/internal/modules/casbin"
	"github.com/ouqiang/gocron/internal/modules/utils"
	"gopkg.in/macaron.v1"
)

// PolicyForm 权限策略表单
type PolicyForm struct {
	Sub    string `binding:"Required;MaxSize(64)"` // 角色/用户
	Domain string `binding:"Required;MaxSize(64)"` // 租户域
	Obj    string `binding:"Required;MaxSize(64)"` // 资源对象
	Act    string `binding:"Required;MaxSize(32)"` // 操作动作
}

// RoleForm 角色分配表单
type RoleForm struct {
	User   string `binding:"Required;MaxSize(64)"`
	Role   string `binding:"Required;MaxSize(64)"`
	Domain string `binding:"Required;MaxSize(64)"`
}

// ListPolicies 获取所有策略规则
func ListPolicies(ctx *macaron.Context) string {
	json := utils.JsonResponse{}
	policies := casbinauth.GetAllPolicies()
	groupings := casbinauth.GetAllGroupingPolicies()
	return json.Success(utils.SuccessContent, map[string]interface{}{
		"policies":  policies,
		"groupings": groupings,
	})
}

// AddPolicy 新增策略规则
func AddPolicy(ctx *macaron.Context) string {
	json := utils.JsonResponse{}
	sub := ctx.QueryTrim("sub")
	dom := ctx.QueryTrim("dom")
	obj := ctx.QueryTrim("obj")
	act := ctx.QueryTrim("act")
	if sub == "" || dom == "" || obj == "" || act == "" {
		return json.CommonFailure("参数不完整")
	}
	err := casbinauth.AddPolicy(sub, dom, obj, act)
	if err != nil {
		return json.CommonFailure("添加策略失败", err)
	}
	_ = casbinauth.SavePolicy()
	return json.Success("添加成功", nil)
}

// RemovePolicy 删除策略规则
func RemovePolicy(ctx *macaron.Context) string {
	json := utils.JsonResponse{}
	sub := ctx.QueryTrim("sub")
	dom := ctx.QueryTrim("dom")
	obj := ctx.QueryTrim("obj")
	act := ctx.QueryTrim("act")
	if sub == "" || dom == "" || obj == "" || act == "" {
		return json.CommonFailure("参数不完整")
	}
	err := casbinauth.RemovePolicy(sub, dom, obj, act)
	if err != nil {
		return json.CommonFailure("删除策略失败", err)
	}
	_ = casbinauth.SavePolicy()
	return json.Success("删除成功", nil)
}

// AddRoleForUser 为用户分配角色
func AddRoleForUser(ctx *macaron.Context) string {
	json := utils.JsonResponse{}
	user := ctx.QueryTrim("user")
	role := ctx.QueryTrim("role")
	domain := ctx.QueryTrim("domain")
	if user == "" || role == "" || domain == "" {
		return json.CommonFailure("参数不完整")
	}
	err := casbinauth.AddRoleForUser(user, role, domain)
	if err != nil {
		return json.CommonFailure("分配角色失败", err)
	}
	_ = casbinauth.SavePolicy()
	return json.Success("分配成功", nil)
}

// RemoveRoleForUser 删除用户角色
func RemoveRoleForUser(ctx *macaron.Context) string {
	json := utils.JsonResponse{}
	user := ctx.QueryTrim("user")
	role := ctx.QueryTrim("role")
	domain := ctx.QueryTrim("domain")
	if user == "" || role == "" || domain == "" {
		return json.CommonFailure("参数不完整")
	}
	err := casbinauth.RemoveRoleForUser(user, role, domain)
	if err != nil {
		return json.CommonFailure("删除角色失败", err)
	}
	_ = casbinauth.SavePolicy()
	return json.Success("删除成功", nil)
}
