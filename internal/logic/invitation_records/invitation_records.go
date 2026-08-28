package recharge_records

import (
	"context"
	"errors"
	"fmt"
	v1 "gov2panel/api/admin/v1"
	"gov2panel/internal/dao"
	d "gov2panel/internal/dao"
	"gov2panel/internal/logic/cornerstone"
	"gov2panel/internal/utils"

	"gov2panel/internal/model/entity"
	"gov2panel/internal/model/model"
	"gov2panel/internal/service"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

type sInvitationRecords struct {
	Cornerstone cornerstone.Cornerstone
}

func init() {
	service.RegisterInvitationRecords(New())
}

func New() *sInvitationRecords {
	return &sInvitationRecords{
		Cornerstone: *cornerstone.NewCornerstoneWithTable(dao.V2InvitationRecords.Table()),
	}
}

// 获取数据
func (s *sInvitationRecords) GetInvitationRecordsList(req *v1.InvitationRecordsReq, orderBy, orderDirection string, offset, limit int) (m []*model.InvitationRecordsInfo, total int, err error) {
	m = make([]*model.InvitationRecordsInfo, 0)
	db := s.Cornerstone.GetDB()

	orderBy = dao.V2InvitationRecords.Table() + "." + orderBy
	db.LeftJoin(
		dao.V2User.Table(),
		"user",
		fmt.Sprintf("%s.%s=user.%s",
			dao.V2InvitationRecords.Table(),
			dao.V2InvitationRecords.Columns().UserId,
			dao.V2User.Columns().Id,
		))
	db.LeftJoin(
		dao.V2User.Table(),
		"from_user",
		fmt.Sprintf("%s.%s=from_user.%s",
			dao.V2InvitationRecords.Table(),
			dao.V2InvitationRecords.Columns().FromUserId,
			dao.V2User.Columns().Id,
		))

	db.WhereLike(fmt.Sprintf("IFNULL(user.%s,\"\")", dao.V2User.Columns().UserName), "%"+req.UserName+"%")
	db.WhereLike(fmt.Sprintf("IFNULL(from_user.%s,\"\")", dao.V2User.Columns().UserName), "%"+req.FromUserName+"%")
	if req.Id != 0 {
		db.Where(dao.V2InvitationRecords.Columns().Id, req.Id)
	}
	if req.UserId != 0 {
		db.Where(dao.V2InvitationRecords.Columns().UserId, req.V2InvitationRecords.UserId)
	}
	if req.FromUserId != 0 {
		db.Where(dao.V2InvitationRecords.Columns().FromUserId, req.V2InvitationRecords.FromUserId)
	}
	if req.OperateType != 0 {
		db.Where(dao.V2InvitationRecords.Columns().OperateType, req.V2InvitationRecords.OperateType)
	}
	if req.State != 0 {
		db.Where(dao.V2InvitationRecords.Columns().State, req.V2InvitationRecords.State)
	}
	if req.RechargeRecordsId != 0 {
		db.Where(dao.V2InvitationRecords.Columns().RechargeRecordsId, req.V2InvitationRecords.RechargeRecordsId)
	}

	dbC := *db
	dbCCount := &dbC

	db.Fields(fmt.Sprintf("%s.*", dao.V2InvitationRecords.Table()))
	err = db.Order(orderBy, orderDirection).Limit(offset, limit).ScanList(&m, "InvitationRecords")
	if err != nil {
		return m, 0, err
	}

	db.Fields("*")
	total, err = dbCCount.Count()
	if err != nil {
		return m, 0, err
	}

	if total > 0 {
		err = s.Cornerstone.GetDBT(d.V2User.Table()).
			Where("id", gdb.ListItemValuesUnique(m, "InvitationRecords", "UserId")).
			ScanList(&m, "User", "InvitationRecords", "id:UserId")

		err = s.Cornerstone.GetDBT(d.V2User.Table()).
			Where("id", gdb.ListItemValuesUnique(m, "InvitationRecords", "FromUserId")).
			ScanList(&m, "FromUser", "InvitationRecords", "id:FromUserId")
	}

	return m, total, err
}

// 获取数据根据用户id
func (s *sInvitationRecords) GetInvitationRecordsListByUserId(userId int, orderBy, orderDirection string, offset, limit int) (m []*model.InvitationRecordsInfo, total int, err error) {
	m = make([]*model.InvitationRecordsInfo, 0)

	db := s.Cornerstone.GetDB()
	db.Where(dao.V2InvitationRecords.Columns().UserId, userId)

	dbC := *db
	dbCCount := &dbC

	err = db.Order(orderBy, orderDirection).Limit(offset, limit).ScanList(&m, "InvitationRecords")
	if err != nil {
		return m, 0, err
	}

	total, err = dbCCount.Count()
	if err != nil {
		return m, 0, err
	}

	if total > 0 {
		err = s.Cornerstone.GetDBT(d.V2User.Table()).Fields(d.V2User.Columns().Id, d.V2User.Columns().UserName).
			Where("id", gdb.ListItemValuesUnique(m, "InvitationRecords", "FromUserId")).
			ScanList(&m, "FromUser", "InvitationRecords", "id:FromUserId")
	}

	for i := 0; i < len(m); i++ {
		if m[i].FromUser != nil {
			m[i].FromUser.UserName = utils.MaskString(m[i].FromUser.UserName)
		}

	}
	return m, total, err
}

// 获取数据根据id
func (s *sInvitationRecords) GetOneById(id int) (d *entity.V2InvitationRecords, err error) {
	err = s.Cornerstone.GetDB().Where(dao.V2InvitationRecords.Columns().Id, id).Scan(&d)
	return
}

// 获取数据根据id
func (s *sInvitationRecords) GetOneByFromUserId(from_user_id int) (d *entity.V2InvitationRecords, err error) {
	err = s.Cornerstone.GetDB().Where(dao.V2InvitationRecords.Columns().FromUserId, from_user_id).Scan(&d)
	return
}

// 添加
func (s *sInvitationRecords) Insert(data *entity.V2InvitationRecords) (err error) {
	err = s.Cornerstone.Save(data)
	return
}

// 更新
func (s *sInvitationRecords) UpInvitationRecordsStateById(id, state int) (err error) {
	_, err = s.Cornerstone.GetDB().Data(g.Map{dao.V2InvitationRecords.Columns().State: state}).WhereIn(dao.V2InvitationRecords.Columns().Id, id).Update()
	return
}

// 审核状态
func (s *sInvitationRecords) AdminiUpStateById(id, state int) (err error) {
	ir, err := s.GetOneById(id)
	if err != nil {
		return err
	}
	if ir.State == state {
		return errors.New("请勿审核一样的状态")
	}

	err = g.DB().Transaction(context.TODO(), func(ctx context.Context, tx gdb.TX) error {
		//检查修改的数据是否有最新的提现记录， 有不让修改
		tjCount, err := tx.Ctx(ctx).Model(d.V2InvitationRecords.Table()).
			WhereGT(d.V2InvitationRecords.Columns().Id, id).
			Where(d.V2InvitationRecords.Columns().UserId, ir.UserId).
			Where(d.V2InvitationRecords.Columns().State, 1).
			Where(d.V2InvitationRecords.Columns().OperateType, 2).Count()
		if err != nil {
			return err
		}
		if tjCount > 0 && ir.State == 1 {
			return errors.New("已提现无法审核")
		}

		if ir.OperateType == 1 { //如果是邀请
			switch state {
			case 1: //审核 直接给用户加佣金
				//给用户加佣金
				_, err = tx.Ctx(ctx).Model(d.V2User.Table()).Where(d.V2User.Columns().Id, ir.UserId).Increment(d.V2User.Columns().CommissionBalance, ir.Amount)
				if err != nil {
					return err
				}

			case 2: //拒绝
				//如果原来已经审核再拒绝，则扣金额;
				if ir.State == 1 {
					//给用户扣佣金
					_, err = tx.Ctx(ctx).Model(d.V2User.Table()).Where(d.V2User.Columns().Id, ir.UserId).Decrement(d.V2User.Columns().CommissionBalance, ir.Amount)
					if err != nil {
						return err
					}
				}

			}
		}

		//更新审核
		_, err = tx.Ctx(ctx).Model(d.V2InvitationRecords.Table()).
			Data(g.Map{d.V2InvitationRecords.Columns().State: state}).
			Where(d.V2InvitationRecords.Columns().Id, id).Update()
		if err != nil {
			return err
		}
		return nil
	})

	return err
}

// CommissionTransferBalance佣金转余额
func (s *sInvitationRecords) CommissionTransferBalance(ctx context.Context, userId int) (err error) {
	return g.DB().Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		// 1. 事务内加排他锁查询最新用户数据，防止并发问题
		var user *entity.V2User
		err := tx.Model(d.V2User.Table()).
			Where(d.V2User.Columns().Id, userId).
			LockUpdate().
			Scan(&user)
		if err != nil {
			return err
		}

		if user == nil {
			return errors.New("用户不存在")
		}

		// 2. 严格校验金额
		transferAmount := user.CommissionBalance
		if transferAmount <= 0 {
			return errors.New("佣金余额不足，无法转入")
		}

		// 3. 写入充值/转换记录
		rr := &entity.V2RechargeRecords{
			Amount:          transferAmount,
			UserId:          user.Id,
			OperateType:     1,
			RechargeName:    "佣金转余额",
			ConsumptionName: "",
			Remarks:         "",
			// 注意：若 transferAmount 为 float，建议格式化为固定 2 位小数的字符串，如 fmt.Sprintf("%.2f", transferAmount)
			TransactionId: utils.RechargeOrderNo(fmt.Sprintf("%.2f", transferAmount), fmt.Sprintf("%.2f", transferAmount), 0, userId),
		}
		if _, err = tx.Model(d.V2RechargeRecords.Table()).Save(rr); err != nil {
			return err
		}

		// 4. 原子更新：清空佣金并同时累加余额（带有条件比较，防并发超扣）
		res, err := tx.Model(d.V2User.Table()).
			Where(d.V2User.Columns().Id, user.Id).
			Where(d.V2User.Columns().CommissionBalance, transferAmount). // 条件匹配防并发
			Data(g.Map{
				d.V2User.Columns().CommissionBalance: 0,
				d.V2User.Columns().Balance:           gdb.Raw(fmt.Sprintf("%s + %v", d.V2User.Columns().Balance, transferAmount)),
			}).
			Update()
		if err != nil {
			return err
		}

		affected, _ := res.RowsAffected()
		if affected == 0 {
			return errors.New("操作失败，账户金额发生变动，请重试")
		}

		return nil
	})
}

// 佣金提现
// 佣金提现
func (s *sInvitationRecords) WithdrawalBalance(ctx context.Context, userId int) (err error) {
	user, err := service.User().GetUserByIdAndCheck(userId)
	if err != nil {
		return err
	}

	setting, err := service.Setting().GetSettingAllMap()
	if err != nil {
		return err
	}

	minBalance := setting["minimum_withdrawal_balance"].Float64()
	if user.CommissionBalance < minBalance {
		return errors.New("佣金低于最低提现额度，最低提现额度：" + setting["minimum_withdrawal_balance"].String())
	}

	// 接收事务执行的 error
	err = g.DB().Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {

		// 1. 原子更新：只有当余额大于等于最低限制且满足原余额时才清零
		// 这样即使有多个并发请求，也只有一个请求能够成功 Update 影响行数大于 0
		res, err := tx.Ctx(ctx).
			Model(d.V2User.Table()).
			Data(g.Map{
				d.V2User.Columns().CommissionBalance: 0,
			}).
			Where(d.V2User.Columns().Id, user.Id).
			Where(d.V2User.Columns().CommissionBalance+" >= ?", minBalance). // 条件锁定
			Update()

		if err != nil {
			return err
		}

		affected, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if affected == 0 {
			// 说明余额已被其他并发请求扣减，直接阻断
			return errors.New("提现失败，余额不足或已被处理")
		}

		// 2. 写入邀请收入扣减记录
		ir := &entity.V2InvitationRecords{
			Amount:            -user.CommissionBalance,
			UserId:            user.Id,
			FromUserId:        0,
			CommissionRate:    0,
			RechargeRecordsId: 0,
			OperateType:       2,
			State:             -1,
		}
		_, err = tx.Ctx(ctx).Model(d.V2InvitationRecords.Table()).Save(ir)
		if err != nil {
			return err
		}

		// 3. 创建提现工单
		_, err = tx.Ctx(ctx).
			Model(d.V2Ticket.Table()).
			Save(&entity.V2Ticket{
				UserId:      user.Id,
				Subject:     "[提现工单]",
				Level:       3,
				Status:      -1,
				ReplyStatus: -1,
			})
		if err != nil {
			return err
		}

		return nil
	})

	return err
}
