package list

import (
	"fmt"
	"github.com/Jinnrry/pmail/consts"
	"github.com/Jinnrry/pmail/db"
	"github.com/Jinnrry/pmail/dto"
	"github.com/Jinnrry/pmail/dto/response"
	"github.com/Jinnrry/pmail/models"
	"github.com/Jinnrry/pmail/utils/array"
	"github.com/Jinnrry/pmail/utils/context"
	log "github.com/sirupsen/logrus"
	"strings"
)

func GetEmailList(ctx *context.Context, tagInfo dto.SearchTag, keyword string, pop3List bool, offset, limit int) (emailList []*response.EmailResponseData, total int64) {
	return getList(ctx, tagInfo, keyword, "all", pop3List, offset, limit)
}

func SearchEmailList(ctx *context.Context, tagInfo dto.SearchTag, keyword, searchField string, pop3List bool, offset, limit int) (emailList []*response.EmailResponseData, total int64) {
	return getList(ctx, tagInfo, keyword, searchField, pop3List, offset, limit)
}

func getList(ctx *context.Context, tagInfo dto.SearchTag, keyword, searchField string, pop3List bool, offset, limit int) (emailList []*response.EmailResponseData, total int64) {
	querySQL, queryParams := genSQL(ctx, false, tagInfo, keyword, searchField, pop3List, offset, limit)

	err := db.Instance.SQL(querySQL, queryParams...).Find(&emailList)
	if err != nil {
		log.WithContext(ctx).Errorf("SQL ERROR: %s ,Error:%s", querySQL, err)
	}

	totalSQL, totalParams := genSQL(ctx, true, tagInfo, keyword, searchField, pop3List, offset, limit)

	_, err = db.Instance.SQL(totalSQL, totalParams...).Get(&total)
	if err != nil {
		log.WithContext(ctx).Errorf("SQL ERROR: %s ,Error:%s", querySQL, err)
	}

	return emailList, total
}

func genSQL(ctx *context.Context, count bool, tagInfo dto.SearchTag, keyword, searchField string, pop3List bool, offset, limit int) (string, []any) {
	sqlParams := []any{ctx.UserID}
	sql := "select "

	if count {
		sql += `count(1) from email e left join user_email ue on e.id=ue.email_id where ue.user_id = ? `
	} else if pop3List {
		sql += `e.id,e.size from email e left join user_email ue on e.id=ue.email_id where ue.user_id = ? `
	} else {
		sql += `e.*,ue.is_read from email e left join user_email ue on e.id=ue.email_id where ue.user_id = ? `
	}

	if tagInfo.Status != -1 {
		switch tagInfo.Status {
		case consts.EmailStatusDel:
			sql += " and (ue.status =? or ue.group_id=?) "
			sqlParams = append(sqlParams, tagInfo.Status, models.Deleted)
		case consts.EmailStatusDrafts:
			sql += " and (ue.status =? or ue.group_id=?) "
			sqlParams = append(sqlParams, tagInfo.Status, models.Drafts)
		case consts.EmailStatusJunk:
			sql += " and (ue.status =? or ue.group_id=?) "
			sqlParams = append(sqlParams, tagInfo.Status, models.Junk)
		}

	} else if tagInfo.Status == -1 {
		if tagInfo.Type != 1 {
			sql += " and ue.status = 0"
		} else {
			// 发件箱不展示已删除的邮件
			sql += " and ue.status != 3"
		}
	}

	if tagInfo.Type == consts.EmailTypeReceive {
		sql += " and type =? "
		sqlParams = append(sqlParams, tagInfo.Type)
	} else if tagInfo.Type == consts.EmailTypeSend {
		sql += " and (type =? or ue.group_id=?)"
		sqlParams = append(sqlParams, tagInfo.Type, models.Sent)
	}

	if tagInfo.GroupId != -1 {
		if tagInfo.GroupId > 0 {
			sql += " and ue.group_id=? "
			sqlParams = append(sqlParams, tagInfo.GroupId)
		} else if tagInfo.GroupId == 0 && tagInfo.Status == -1 {
			sql += " and (ue.group_id=? or ue.group_id=0)"
			sqlParams = append(sqlParams, models.INBOX)
		}
	} else {
		sql += " and (ue.group_id=0 or ue.group_id =?) "
		sqlParams = append(sqlParams, models.INBOX)
	}

	quoteColumn := func(column string) string { return column }
	if db.Instance != nil {
		quoteColumn = db.Instance.Quote
	}
	sql, sqlParams = appendKeywordSearch(sql, sqlParams, keyword, searchField, quoteColumn)

	// A count query has exactly one result row. Applying page OFFSET to it
	// drops that row on page 2 and makes the client hide pagination entirely.
	if !count {
		if limit <= 0 {
			limit = 10
		}
		if offset < 0 {
			offset = 0
		}
		if !pop3List && limit > 200 {
			limit = 200
		}
		sql += " order by e.id desc"
		if limit < 10000 {
			sql += fmt.Sprintf(" LIMIT %d OFFSET %d ", limit, offset)
		}
	}

	return sql, sqlParams

}

func appendKeywordSearch(sql string, sqlParams []any, keyword, searchField string, quoteColumn func(string) string) (string, []any) {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return sql, sqlParams
	}

	recipientColumns := []string{"to", "cc", "bcc"}
	columns := recipientColumns
	if searchField != "recipient" {
		columns = []string{"subject", "text", "html", "from_address", "from_name", "to", "cc", "bcc"}
	}

	conditions := make([]string, 0, len(columns))
	// '_' and '%' are legal mailbox characters, not search wildcards.
	keyword = strings.NewReplacer("!", "!!", "%", "!%", "_", "!_").Replace(keyword)
	pattern := "%" + keyword + "%"
	for _, column := range columns {
		conditions = append(conditions, "e."+quoteColumn(column)+" like ? escape '!'")
		sqlParams = append(sqlParams, pattern)
	}
	return sql + " and (" + strings.Join(conditions, " or ") + ")", sqlParams
}

type statRes struct {
	Total int64
	Size  int64
}

// Stat 查询邮件总数和大小
func Stat(ctx *context.Context) (int64, int64) {
	sql := `select count(1) as total,sum(size) as size from email e left join user_email ue on e.id=ue.email_id where ue.user_id = ? and e.type = 0 and ue.status != 3`
	var ret statRes
	_, err := db.Instance.SQL(sql, ctx.UserID).Get(&ret)
	if err != nil {
		log.WithContext(ctx).Errorf("SQL ERROR: %s ,Error:%s", sql, err)
	}
	return ret.Total, ret.Size
}

type ImapListReq struct {
	UidList []int
	Star    int
	End     int
}

func GetUEListByUID(ctx *context.Context, groupName string, start, end int, uidList []int) []*response.UserEmailUIDData {
	var rows []*response.UserEmailUIDData
	query := db.Instance.Table("user_email").Where("user_id=?", ctx.UserID)
	if code, ok := models.GroupNameToCode[groupName]; ok {
		status := map[int]int{models.INBOX: 0, models.Sent: 1, models.Drafts: 4, models.Deleted: 3, models.Junk: 5}[code]
		query = query.And("status=? and group_id=0", status)
	} else {
		var mailbox models.Group
		found, err := db.Instance.Where("user_id=? and full_path=?", ctx.UserID, groupName).Get(&mailbox)
		if err != nil || !found {
			return nil
		}
		query = query.And("group_id=?", mailbox.ID)
	}
	if err := query.Select("*").OrderBy("id").Find(&rows); err != nil {
		log.WithContext(ctx).Errorf("IMAP mailbox query: %v", err)
		return nil
	}
	var result []*response.UserEmailUIDData
	for index, row := range rows {
		// Sequence numbers belong to the complete selected mailbox, not the
		// subset selected by UID or search criteria.
		row.SerialNumber = index + 1
		if start > 0 && row.ID < start || end > 0 && row.ID > end {
			continue
		}
		if len(uidList) > 0 && !array.InArray(row.ID, uidList) {
			continue
		}
		result = append(result, row)
	}
	return result
}

func GetEmailListByGroup(ctx *context.Context, groupName string, req ImapListReq, uid bool) []*response.EmailResponseData {
	rows := GetUEListByUID(ctx, groupName, 0, 0, nil)
	if len(rows) == 0 {
		return nil
	}
	maximum := len(rows)
	if uid {
		maximum = rows[len(rows)-1].ID
	}
	start, end := req.Star, req.End
	// IMAP represents "*" as zero and permits reversed ranges, including *:1.
	if start == 0 {
		start = maximum
	}
	if end == 0 {
		end = maximum
	}
	if start > end {
		start, end = end, start
	}
	var selected []*response.UserEmailUIDData
	for _, row := range rows {
		number := row.SerialNumber
		if uid {
			number = row.ID
		}
		matches := number >= start && number <= end
		if len(req.UidList) > 0 {
			matches = array.InArray(number, req.UidList)
		}
		if matches {
			selected = append(selected, row)
		}
	}
	return GetEmailsForUIDRows(ctx, selected)
}

// GetEmailsForUIDRows materializes only previously authorized mailbox rows.
func GetEmailsForUIDRows(ctx *context.Context, selected []*response.UserEmailUIDData) []*response.EmailResponseData {
	var emailIDs []int
	for _, row := range selected {
		if row.UserID == ctx.UserID {
			emailIDs = append(emailIDs, row.EmailID)
		}
	}
	if len(emailIDs) == 0 {
		return nil
	}
	var emails []models.Email
	if err := db.Instance.In("id", emailIDs).Find(&emails); err != nil {
		log.WithContext(ctx).Errorf("IMAP message query: %v", err)
		return nil
	}
	byID := make(map[int]models.Email, len(emails))
	for _, email := range emails {
		byID[email.Id] = email
	}
	var result []*response.EmailResponseData
	for _, row := range selected {
		if row.UserID != ctx.UserID {
			continue
		}
		email, ok := byID[row.EmailID]
		if !ok {
			continue
		}
		result = append(result, &response.EmailResponseData{
			Email: email, IsRead: row.IsRead, UeId: row.ID, SerialNumber: row.SerialNumber,
		})
	}
	return result
}
