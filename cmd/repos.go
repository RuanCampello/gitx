package cmd

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/charmbracelet/x/term"
	"github.com/spf13/cobra"
)

type repository struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Language    string    `json:"language"`
	Updated     time.Time `json:"updated_at"`
	Url         string    `json:"svn_url"`
	Fork        bool      `json:"fork"`
}

var reposCmd = &cobra.Command{
	Use:   "repos [username]",
	Short: "List the (public) repositories of a GitHub user",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		username := args[0]
		maxResults, _ := cmd.Flags().GetInt("number")
		language, _ := cmd.Flags().GetString("language")

		repos, err := getRepos(username, language, maxResults)
		if err != nil {
			fmt.Println("Error during repository fetch", err)
			os.Exit(0)
		}
		table, err := renderRepos(repos)
		if err != nil {
			fmt.Println("Error rendering repositories table", err)
			os.Exit(0)
		}

		fmt.Println(table)
	},
}

func init() {
	gitxCmd.AddCommand(reposCmd)
	reposCmd.Flags().IntP("number", "n", 5, "Maximum number of repositories to list")
	reposCmd.Flags().StringP("language", "l", "all", "The languange of the repository")
}

func getRepos(username, lang string, max int) ([]repository, error) {
	url := fmt.Sprintf("https://api.github.com/users/%s/repos?per_page=%d&sort=update", username, max)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch repositories for user %s: %s", username, resp.Status)
	}

	var repositories []repository
	if err := json.NewDecoder(resp.Body).Decode(&repositories); err != nil {
		return nil, err
	}

	var filteredRepos []repository

	for _, repo := range repositories {
		if lang != "all" {
			if !repo.Fork && strings.ToLower(repo.Language) == lang {
				filteredRepos = append(filteredRepos, repo)
			}
		} else if !repo.Fork {
			filteredRepos = append(filteredRepos, repo)
		}
	}

	return filteredRepos, nil
}

// func BubbleSortRepos(repos []repository) {
// 	n := len(repos)
// 	for i := 0; i < n-1; i++ {
// 		for j := 0; j < n-i-1; j++ {
// 			if repos[j].Updated.Before(repos[j+1].Updated) {
// 				repos[j], repos[j+1] = repos[j+1], repos[j]
// 			}
// 		}
// 	}
// }

func renderRepos(repos []repository) (string, error) {
	headers := []string{"Name", "Description", "Url", "Last update"}
	// BubbleSortRepos(repos)
	rows := make([][]string, len(repos))
	for i, repo := range repos {
		rows[i] = []string{
			repo.Name,
			repo.Description,
			repo.Url,
			repo.Updated.Format("02-01-2006"),
		}
	}

	width, height, err := term.GetSize(os.Stdout.Fd())
	if err != nil {
		return "", err
	}
	re := lipgloss.NewRenderer(os.Stdout)

	t := table.New().Rows(rows...).Headers(headers...).Width(width - (int(math.Pow(float64(width), 0.2)))).
		BorderStyle(re.NewStyle().Foreground(lipgloss.Color("92")))

	return lipgloss.Place(
		width,
		height/2,
		lipgloss.Center,
		lipgloss.Center,
		lipgloss.JoinVertical(lipgloss.Center, t.Render()),
	), nil
}
