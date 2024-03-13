package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

const layout = "2006-01-02 15:04:05"
const startTimeSuffix = " 00:00:00"
const endTimeSuffix = " 23:59:59"

func main() {
	var t = flag.String("pat", "hoge", "personal access token")
	var s = flag.String("s", "", "start date")
	var e = flag.String("e", "", "end date")
	var repo = flag.String("repo", "", "repository")
	flag.Parse()
	var personalAccessToken = *t
	var startTimeString = *s
	var endTimeString = *e
	startTime, endTime := getPeriod(startTimeString, endTimeString)
	ctx := context.Background()

	//認証を行う
	fmt.Println("authentication start")
	client := GetAuthenticatedClient(personalAccessToken, ctx)

	repositoryList := []string{*repo}

	//pullRequestの一覧を取得
	fmt.Println("getting pullRequest list start")
	var pullRequestList []*github.PullRequest
	for _, repository := range repositoryList {
		org := "demae-can"
		tmpPullRequestList := GetPullRequestList(client, ctx, org, repository, startTime, endTime)
		pullRequestList = append(pullRequestList, tmpPullRequestList...)
	}

	userPrCountMap := map[string]int32{}
	userPrLeadTimeMap := map[string]float64{}

	for _, pullRequest := range pullRequestList {
		user := pullRequest.GetUser().GetLogin()
		leadTimeDuration := pullRequest.GetMergedAt().Sub(pullRequest.GetCreatedAt())

		userPrCountMap[user] += 1
		userPrLeadTimeMap[user] += leadTimeDuration.Abs().Seconds()
	}
	sumPrCount := int32(0)
	sumPrLeadTime := 0.
	for user, prCount := range userPrCountMap {
		sumPrCount += prCount
		sumPrLeadTime += userPrLeadTimeMap[user]
		prLeadTimeAverage := userPrLeadTimeMap[user] / float64(prCount)
		fmt.Println(user, prCount, time.Second*time.Duration(prLeadTimeAverage))
	}
	fmt.Println("pr count", sumPrCount)
	fmt.Println("pr leadtime", time.Second*time.Duration(sumPrLeadTime/float64(sumPrCount)))
}

func getPeriod(startTimeString string, endTimeString string) (time.Time, time.Time) {
	location, _ := time.LoadLocation("Asia/Tokyo")
	startTime, _ := time.ParseInLocation(layout, startTimeString+startTimeSuffix, location)
	endTime, _ := time.ParseInLocation(layout, endTimeString+endTimeSuffix, location)
	fmt.Println(startTime, endTime)
	return startTime, endTime
}

func GetAuthenticatedClient(accessToken string, ctx context.Context) *github.Client {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: accessToken},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)
	return client
}

func GetPullRequestList(client *github.Client, ctx context.Context, org string, repo string, startTime time.Time, endTime time.Time) []*github.PullRequest {

	opt := &github.PullRequestListOptions{
		State:       "all",
		ListOptions: github.ListOptions{PerPage: 30},
		Sort:        "created",
		Direction:   "desc",
	}

	var pullRequestList []*github.PullRequest
	for {
		tmpPullRequestList, response, err := client.PullRequests.List(ctx, org, repo, opt)
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
		for _, pullRequest := range tmpPullRequestList {
			if pullRequest.GetCreatedAt().After(endTime) {
				continue
			}
			if pullRequest.GetUpdatedAt().Before(startTime) {
				continue
			}
			if pullRequest.MergedAt == nil {
				continue
			}
			//debug
			// fmt.Println(
			// 	org,
			// 	repo,
			// 	pullRequest.GetNumber(),
			// 	pullRequest.Assignee.GetLogin(),
			// 	pullRequest.GetCreatedAt(),
			// 	pullRequest.GetMergedAt(),
			// )
			pullRequestList = append(pullRequestList, pullRequest)
		}

		time.Sleep(500 * time.Millisecond)

		if response.NextPage == 0 {
			break
		}
		opt.Page = response.NextPage
	}

	return pullRequestList
}
