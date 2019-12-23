package pojo

import "tesou.io/platform/foot-parent/foot-api/common/base/pojo"

type AsiaHis struct {
	/**
	初上下盘赔率
	*/
	Sp3 float64
	Sp0 float64
	//让球
	SLetBall float64	`xorm:" comment('s让球') index"`

	/**
	即时上下盘赔率
	*/
	Ep3 float64
	Ep0 float64
	//让球
	ELetBall float64 `xorm:" comment('e让球') index"`

	//博彩公司id
	CompId string `xorm:"unique(CompId_MatchId_OddDate)"`
	//比赛id
	MatchId string `xorm:"unique(CompId_MatchId_OddDate)"`
	//数据时间
	OddDate string	`xorm:"unique(CompId_MatchId_OddDate)"`

	pojo.BasePojo `xorm:"extends"`
}
