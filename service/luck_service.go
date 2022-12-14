package service

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"sale-service/api"
	"sale-service/constant"
	"sale-service/dao"
	"sale-service/engine"
	"sale-service/model"
	"strings"

	"github.com/jinzhu/copier"
	"github.com/spf13/cast"
)

type LuckService struct {
}

var LuckServiceInstance *LuckService

func init() {
	LuckServiceInstance = &LuckService{}
}

func (ls *LuckService) AddLuck(luck *model.Luck) *api.JsonResult {
	luckDao := &dao.LuckDao{Db: engine.GetOrmEngine()}
	result, err := luckDao.InsertLuck(luck)
	if err != nil {
		return api.JsonError(api.AddLuckFailErr).JsonWithMsg(err.Error())
	}

	return api.JsonSuccess().JsonWithData(result)
}

func (ls *LuckService) GetLuck(id int64) *api.JsonResult {
	luckDao := &dao.LuckDao{Db: engine.GetOrmEngine()}
	luck := &model.Luck{Id: id, Status: constant.STATUS_ENABLE}
	result, err := luckDao.Db.Get(luck)
	if err != nil {
		return api.JsonError(api.GetLuckFailErr).JsonWithMsg(err.Error())
	}
	if !result {
		return api.JsonError(api.GetLuckFailErr).JsonWithMsg(api.GetLuckFailErr.Message)
	}

	items := make([]*model.LuckItem, 0)
	item := &model.LuckItem{LuckId: id, Status: constant.STATUS_ENABLE}
	err = luckDao.Db.Find(&items, item)
	if err != nil {
		return api.JsonError(api.GetLuckFailErr).JsonWithMsg(err.Error())
	}
	luckResponse := &model.LuckResponse{}
	copier.Copy(&luckResponse, &luck)
	luckResponse.Items = items
	return api.JsonSuccess().JsonWithData(luckResponse)
}

func (ls *LuckService) AddDraw(luck_id int64, user_name string) *api.JsonResult {
	luck := ls.GetLuck(luck_id)
	if luck.Code != 200 {
		return luck
	}
	luckDao := &dao.LuckDao{Db: engine.GetOrmEngine()}

	//获取用户信息
	luckUser := &model.LuckUser{Username: user_name}
	res, err := luckDao.Db.Get(luckUser)
	if err != nil {
		return api.JsonError(api.ErrServer).JsonWithMsg(err.Error())
	}

	//判断是否已经抽过奖
	if res {
		luckRecord := &model.LuckRecord{UserId: luckUser.Id, LuckId: luck_id}
		res, err := luckDao.Db.Get(luckRecord)
		if err != nil {
			return api.JsonError(api.ErrServer).JsonWithMsg(err.Error())
		}
		if res {
			return api.JsonError(api.LuckRepeatErr)
		}
	}

	luckResponse := &model.LuckResponse{}
	copier.Copy(&luckResponse, luck.Data)
	index := ls.violence(luckResponse.Items)
	item := luckResponse.Items[index]
	if item.Quantity <= 0 {
		return api.JsonError(api.LuckQuantityErr)
	}
	record := &model.LuckRecord{ItemId: item.Id, LuckId: luck_id}
	err = luckDao.AddDraw(item, record, luckUser)
	if err != nil {
		return api.JsonError(api.AddLuckFailErr).JsonWithMsg(err.Error())
	}
	return api.JsonSuccess().JsonWithData(item)
}

//根据username获取user信息
func (ls *LuckService) UpdateUserPhone(username string, phone string) *api.JsonResult {
	luckDao := &dao.LuckDao{Db: engine.GetOrmEngine()}
	luckUser := &model.LuckUser{Mobile: phone, Username: username}
	res, err := luckDao.UpdateUserPhone(luckUser)
	if err != nil {
		return api.JsonError(api.LuckUpdatePhoneErr).JsonWithMsg(err.Error())
	}
	return api.JsonSuccess().JsonWithData(res)
}

// 抽奖
func (ls *LuckService) violence(items []*model.LuckItem) int {
	length := 0
	data := ""
	for index, value := range items {
		length += value.Probability
		position := fmt.Sprintf("%d,", index)
		data += strings.Repeat(position, value.Probability)
	}

	// 获取[1,0) 随机数
	res, _ := rand.Int(rand.Reader, big.NewInt(int64(length)))
	index := res.Int64()

	arr := strings.Split(data, ",")
	giftIndex := cast.ToInt(arr[index])
	return giftIndex
}
