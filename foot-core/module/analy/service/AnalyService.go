package service

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"tesou.io/platform/foot-parent/foot-api/common/base"
	entity5 "tesou.io/platform/foot-parent/foot-api/module/analy/pojo"
	"tesou.io/platform/foot-parent/foot-api/module/analy/vo"
	entity2 "tesou.io/platform/foot-parent/foot-api/module/match/pojo"
	entity3 "tesou.io/platform/foot-parent/foot-api/module/odds/pojo"
	"tesou.io/platform/foot-parent/foot-core/common/base/service/mysql"
	"tesou.io/platform/foot-parent/foot-core/common/utils"
	"tesou.io/platform/foot-parent/foot-core/module/analy/constants"
	service3 "tesou.io/platform/foot-parent/foot-core/module/elem/service"
	service2 "tesou.io/platform/foot-parent/foot-core/module/match/service"
	"tesou.io/platform/foot-parent/foot-core/module/odds/service"
	"time"
)

type AnalyService struct {
	mysql.BaseService
	service.EuroLastService
	service.EuroHisService
	service.AsiaLastService
	service2.MatchLastService
	service2.MatchHisService
	service3.LeagueService
	//是否打印赔率数据
	PrintOddData bool
}

func (this *AnalyService) Find(matchId string, alFlag string) *entity5.AnalyResult {
	data := entity5.AnalyResult{MatchId: matchId, AlFlag: alFlag}
	mysql.GetEngine().Get(&data)
	return &data
}

func (this *AnalyService) FindAll() []*entity5.AnalyResult {
	dataList := make([]*entity5.AnalyResult, 0)
	mysql.GetEngine().OrderBy("CreateTime Desc").Find(&dataList)
	return dataList
}

/**
###推送的主客队选项,
#格式为:时间:选项,时间:选项,时间:选项
#时间只支持设置小时数
#3 只推送主队, 1 只推送平局, 0 只推送客队,-1 全部推送
#示例0-3:-1,4-19:3,19-23:-1,未设置时间段为默认只推送3
*/
func (this *AnalyService) teamOption() int {
	var result int
	tempOptionConfig := utils.GetVal(constants.SECTION_NAME, "team_option")
	if len(tempOptionConfig) <= 0 {
		//默认返回 主队选项
		return 3
	}
	//当前的小时
	currentHour, _ := strconv.Atoi(time.Now().Format("15"))
	hourRange_options := strings.Split(tempOptionConfig, ",")
	for _, e := range hourRange_options {
		h_o := strings.Split(e, ":")
		hourRanges := strings.Split(h_o[0], "-")
		option, _ := strconv.Atoi(h_o[1])
		hourBegin, _ := strconv.Atoi(hourRanges[0])
		hourEnd, _ := strconv.Atoi(hourRanges[1])
		if hourBegin <= currentHour && currentHour <= hourEnd {
			result = option
			break;
		}
	}
	return result
}

func (this *AnalyService) ModifyResult() {
	sql_build := `
SELECT 
  ar.* 
FROM
  foot.t_analy_result ar 
WHERE ar.MatchDate < NOW() 
  AND ar.Result = '待定' 
     `
	//结果值
	entitys := make([]*entity5.AnalyResult, 0)
	//执行查询
	this.FindBySQL(sql_build, &entitys)

	if len(entitys) <= 0 {
		return
	}
	for _, e := range entitys {
		aList := this.AsiaLastService.FindByMatchIdCompId(e.MatchId, "澳门")
		if nil == aList || len(aList) < 1 {
			aList = make([]*entity3.AsiaLast,1)
			aList[0] = new(entity3.AsiaLast)
		}
		his := this.MatchHisService.FindById(e.MatchId)
		if nil == his {
			continue
		}
		last := new(entity2.MatchLast)
		last.MatchDate = his.MatchDate
		last.DataDate = his.DataDate
		last.LeagueId = his.LeagueId
		last.MainTeamId = his.MainTeamId
		last.MainTeamGoals = his.MainTeamGoals
		last.GuestTeamId = his.GuestTeamId
		last.GuestTeamGoals = his.GuestTeamGoals
		e.Result = this.IsRight(aList[0], last, e)
		this.Modify(e)
	}

}

func (this *AnalyService) ListDefaultData() []*vo.AnalyResultVO {
	teamOption := this.teamOption()
	al_flag := utils.GetVal(constants.SECTION_NAME, "al_flag")
	hit_count_str := utils.GetVal(constants.SECTION_NAME, "hit_count")
	hit_count, _ := strconv.Atoi(hit_count_str)
	//获取分析计算出的比赛列表
	analyList := this.ListData(al_flag, hit_count, teamOption)
	return analyList
}

/**
获取可发布的数据项
1.预算结果是主队
2.比赛未开始
3.比赛未结束
4.alName 算法名称，默认为
5.option 3(只筛选主队),1(只筛选平局),0(只筛选客队)选项
*/
func (this *AnalyService) ListData(alName string, hitCount int, option int) []*vo.AnalyResultVO {
	sql_build := `
SELECT 
  l.Name as LeagueName,
  ml.MainTeamId,
  ml.GuestTeamId,
  ar.* 
FROM
  foot.t_match_last ml,
  foot.t_league l,
  foot.t_analy_result ar 
WHERE ml.LeagueId = l.Id 
  AND ml.Id = ar.MatchId 
  AND ar.HitCount >= THitCount
  AND ar.LeisuPubd IS FALSE 
  AND ar.MatchDate > NOW()
     `

	if len(alName) > 0 {
		sql_build += " AND ar.AlFlag = '" + alName + "' "
	}
	if hitCount > 0 {
		sql_build += " AND ar.HitCount >= " + strconv.Itoa(hitCount)
	}
	if option >= 0 {
		sql_build += " AND ar.PreResult = " + strconv.Itoa(option) + " "
	}
	sql_build += " ORDER BY ar.MatchDate ASC ,ar.PreResult DESC  "
	//结果值
	entitys := make([]*vo.AnalyResultVO, 0)
	//执行查询
	this.FindBySQL(sql_build, &entitys)
	return entitys
}

//测试加载数据
func (this *AnalyService) LoadData(matchId string) []*entity5.AnalyResult {
	sql_build := `
SELECT 
  ml.*,
  bc.id,
  bc.name AS compName,
  el.* 
FROM
  t_match_last ml,
  t_euro_last el,
  t_comp bc 
WHERE ml.id = el.matchid 
  AND el.compid = bc.id 
	`
	sql_build += "  AND ml.id = '" + matchId + "' "
	//结果值
	entitys := make([]*entity5.AnalyResult, 0)
	//执行查询
	this.FindBySQL(sql_build, &entitys)
	return entitys
}

func (this *AnalyService) IsRight(last *entity3.AsiaLast, v *entity2.MatchLast, analy *entity5.AnalyResult) string {
	//比赛结果
	globalResult := this.ActualResult(last, v,analy)
	var resultFlag string
	if globalResult == -1 {
		resultFlag = "待定"
	} else if globalResult == analy.PreResult {
		resultFlag = "正确"
	} else if globalResult == 1 {
		resultFlag = "走盘"
	} else {
		resultFlag = "错误"
	}

	//打印数据
	league := this.LeagueService.FindById(v.LeagueId)
	matchDate := v.MatchDate.Format("2006-01-02 15:04:05")
	base.Log.Info("比赛Id:" + v.Id + ",比赛时间:" + matchDate + ",联赛:" + league.Name + ",对阵:" + v.MainTeamId + "(" + strconv.FormatFloat(last.ELetBall, 'f', -1, 64) + ")" + v.GuestTeamId + ",预算结果:" + strconv.Itoa(analy.PreResult) + ",已得结果:" + strconv.Itoa(v.MainTeamGoals) + "-" + strconv.Itoa(v.GuestTeamGoals) + " (" + resultFlag + ")")
	return resultFlag
}

/**
比赛的实际结果计算
*/
func (this *AnalyService) ActualResult(last *entity3.AsiaLast, v *entity2.MatchLast, analy *entity5.AnalyResult) int {
	var result int
	h2, _ := time.ParseDuration("148m")
	matchDate := v.MatchDate.Add(h2)
	if matchDate.After(time.Now()) {
		//比赛未结束
		return -1
	}

	var elb_sum float64
	if analy.LetBall > 0 {
		elb_sum = analy.LetBall
	}else{
		elb_sum = last.ELetBall
	}
	var mainTeamGoals float64
	if elb_sum > 0 {
		mainTeamGoals = float64(v.MainTeamGoals) - elb_sum
	} else {
		mainTeamGoals = float64(v.MainTeamGoals) + math.Abs(elb_sum)
	}
	//diff_goals := float64(v.MainTeamGoals-v.GuestTeamGoals) - elb_sum
	//if diff_goals <= 0.25 && diff_goals >= -0.25 {
	//	result = 1
	//}
	if mainTeamGoals > float64(v.GuestTeamGoals) {
		result = 3
	} else if mainTeamGoals < float64(v.GuestTeamGoals) {
		result = 0
	} else {
		result = 1
	}
	return result
}

/**
1.欧赔是主降还是主升 主降为true
*/
func EuroMainDown(e1data *entity3.EuroLast, e2data *entity3.EuroLast) int {
	if e1data.Ep3 < e1data.Sp3 && e2data.Ep3 < e2data.Sp3 {
		return 3
	} else if e1data.Ep0 < e1data.Sp0 && e2data.Ep0 < e2data.Sp0 {
		return 0
	}
	return 1
}

/**
2.亚赔是主降还是主升 主降为true
*/
func AsiaMainDown(a1betData *entity3.AsiaLast) bool {
	slb_sum := a1betData.SLetBall
	elb_sum := a1betData.ELetBall

	if elb_sum > slb_sum {
		return true
	} else if elb_sum < slb_sum {
		return false
	} else { //初始让球和即时让球一致
		if a1betData.Ep3 < a1betData.Sp3 {
			return true
		}
	}
	return false
}

func Decimal(value float64) float64 {
	value, _ = strconv.ParseFloat(fmt.Sprintf("%.2f", value), 64)
	return value
}
