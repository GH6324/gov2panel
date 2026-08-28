package plan

import (
	"context"
	"errors"
	"fmt"
	userv1 "gov2panel/api/user/v1"
	"gov2panel/internal/dao"
	d "gov2panel/internal/dao"
	"gov2panel/internal/logic/cornerstone"
	"gov2panel/internal/model/entity"
	"gov2panel/internal/model/model"
	"gov2panel/internal/service"
	"gov2panel/internal/utils"
	"time"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/shopspring/decimal"
)

type sPlan struct {
	Cornerstone cornerstone.Cornerstone
}

func init() {
	service.RegisterPlan(New())
}

func New() *sPlan {
	return &sPlan{
		Cornerstone: *cornerstone.NewCornerstoneWithTable(dao.V2Plan.Table()),
	}
}

// AE设置
func (s *sPlan) AEPlan(data *entity.V2Plan) (err error) {
	if data.Id != 0 {
		err = s.Cornerstone.UpdateById(data.Id, data)
		return
	}

	err = s.Cornerstone.Save(data)
	return
}

// 删除
func (s *sPlan) DelPlan(ids []int) error {
	serverCount, err := service.ProxyService().GetServiceCountByPlanId(ids)
	if err != nil {
		return err
	}

	if serverCount > 0 {
		return errors.New("订阅有节点在使用，无法删除！")
	}

	userCount, err := service.User().GetUserCountByGroupIds(ids)
	if err != nil {
		return err
	}

	if userCount > 0 {
		return errors.New("订阅有用户在使用，无法删除！")
	}

	return s.Cornerstone.DelByIds(ids)
}

// 获取所有
func (s *sPlan) GetPlanAllList(req entity.V2Plan) (m []*entity.V2Plan, err error) {
	m = make([]*entity.V2Plan, 0)
	err = s.Cornerstone.GetDB().
		OmitEmpty().
		Where(dao.V2Plan.Columns().Id, req.Id).
		WhereLike(dao.V2Plan.Columns().Name, "%"+req.Name+"%").
		OrderDesc("order_id").Scan(&m)
	return m, err
}

// 获取显示的订阅
func (s *sPlan) GetPlanShowList() (m []*entity.V2Plan, err error) {
	m = make([]*entity.V2Plan, 0)
	err = s.Cornerstone.GetDB().
		Where(dao.V2Plan.Columns().Show, 1).
		OrderDesc("order_id").Scan(&m)
	return m, err
}

// 获取显示的订阅 可覆盖的
func (s *sPlan) GetPlanShowAndResetTrafficMethod1List() (m []*entity.V2Plan, err error) {
	m = make([]*entity.V2Plan, 0)
	err = s.Cornerstone.GetDB().
		Where(dao.V2Plan.Columns().Show, 1).
		Where(dao.V2Plan.Columns().ResetTrafficMethod, 1).
		OrderDesc("order_id").Scan(&m)
	return m, err
}

// 获取可覆盖的订阅
func (s *sPlan) GetPlanResetTrafficMethod1List() (m []*entity.V2Plan, err error) {
	m = make([]*entity.V2Plan, 0)
	err = s.Cornerstone.GetDB().
		Where(dao.V2Plan.Columns().ResetTrafficMethod, 1).
		OrderDesc("order_id").Scan(&m)
	return m, err
}

// 根据id获取
func (s *sPlan) GetPlanById(id int) (d *entity.V2Plan, err error) {
	d = new(entity.V2Plan)
	err = s.Cornerstone.GetOneById(id, d)
	return
}

// 用户购买/续费套餐处理
func (s *sPlan) UserBuyAndRenew(ctx context.Context, code string, plan *entity.V2Plan) error {
	user := service.User().GetCtxUser(ctx)

	// 1. 使用 decimal 解决精度问题
	price := decimal.NewFromFloat(plan.Price)

	// 检查用户专享折扣
	if user.Discount > 0 {
		discount := decimal.NewFromFloat(user.Discount).Div(decimal.NewFromInt(100))
		price = price.Mul(decimal.NewFromInt(1).Sub(discount))
	}

	// 检查套餐当前用户数量
	if plan.CapacityLimit > 0 && user.GroupId != plan.Id {
		planUserCount, err := service.User().GetUserCountByPlanID(plan.Id)
		if err != nil {
			return err
		}
		if planUserCount >= plan.CapacityLimit {
			return errors.New("当前订阅人数达到上限！")
		}
	}

	var couponRes *userv1.CouponRes
	var err error

	if code != "" {
		couponRes, err = service.Coupon().CheckCouponCanUseByCode(ctx, &userv1.CouponReq{Code: code, PlanId: plan.Id})
		if err != nil {
			return err
		}

		couponValue := decimal.NewFromFloat(couponRes.Data.Value)
		switch couponRes.Data.Type {
		case 1: // 金额优惠
			price = price.Sub(couponValue)
		case 2: // 比例优惠
			discount := couponValue.Div(decimal.NewFromInt(100))
			price = price.Mul(decimal.NewFromInt(1).Sub(discount))
		}
	}

	// 保证最终金额不小于 0，并保留 2 位小数（四舍五入）
	if price.IsNegative() {
		price = decimal.Zero
	} else {
		price = price.RoundFloor(2)
	}

	finalPriceFloat, _ := price.Float64()

	if decimal.NewFromFloat(user.Balance).LessThan(price) {
		return errors.New("余额不足，请去钱包充值")
	}

	// 2. 使用外部传入的 ctx 开启事务
	err = g.DB().Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		// 扣款记录
		err = service.RechargeRecords().SaveRechargeRecords(
			&entity.V2RechargeRecords{
				Amount:          plan.Price,
				UserId:          user.Id,
				OperateType:     2,
				ConsumptionName: plan.Name,
			},
			"",
			finalPriceFloat,
			plan.Id,
			code,
		)
		if err != nil {
			return err
		}

		// 优惠码使用记录
		if code != "" {
			_, err := tx.Ctx(ctx).Insert(d.V2CouponUse.Table(), g.Map{
				d.V2CouponUse.Columns().CouponId: couponRes.Data.Id,
				d.V2CouponUse.Columns().UserId:   user.Id,
				d.V2CouponUse.Columns().PlanId:   plan.Id,
			})
			if err != nil {
				return err
			}
		}

		// 3. 流量与过期时间更新（修复已过期续费的时间计算 Bug）
		var userUpData g.Map
		switch plan.ResetTrafficMethod { // 1 覆盖、2 叠加
		case 1:
			userUpData = g.Map{
				d.V2User.Columns().GroupId:        plan.Id,
				d.V2User.Columns().U:              0,
				d.V2User.Columns().D:              0,
				d.V2User.Columns().TransferEnable: utils.GBToBytes(plan.TransferEnable),
				d.V2User.Columns().ExpiredAt:      time.Now().Add(time.Duration(plan.Expired) * 24 * time.Hour),
			}
		case 2:
			// 如果已经过期，则从当前时间开始加天数，否则在原过期时间上增加
			expiredRaw := fmt.Sprintf(
				"DATE_ADD(IF(%s > NOW(), %s, NOW()), INTERVAL %d DAY)",
				d.V2User.Columns().ExpiredAt,
				d.V2User.Columns().ExpiredAt,
				plan.Expired,
			)
			userUpData = g.Map{
				d.V2User.Columns().TransferEnable: gdb.Raw(fmt.Sprintf("%s + %d", d.V2User.Columns().TransferEnable, utils.GBToBytes(plan.TransferEnable))),
				d.V2User.Columns().ExpiredAt:      gdb.Raw(expiredRaw),
			}
		}

		_, err = tx.Ctx(ctx).Model(d.V2User.Table()).Data(userUpData).Where(d.V2User.Columns().Id, user.Id).Update()
		return err
	})

	if err != nil {
		return err
	}

	// 更新缓存
	updatedUser, _ := service.User().GetUserById(user.Id)
	service.User().MUpUserMap(model.UserToUserTraffic(updatedUser))

	return nil
}
