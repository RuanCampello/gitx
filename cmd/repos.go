package cmd

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/briandowns/spinner"
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

var wg sync.WaitGroup

var reposCmd = &cobra.Command{
	Use:   "repos [username]",
	Short: "List the (public) repositories of a GitHub user",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		username := args[0]
		maxResults, _ := cmd.Flags().GetInt("number")
		language, _ := cmd.Flags().GetString("language")

		s := spinner.New(spinner.CharSets[35], 50*time.Millisecond)
		process := make(chan string)
		wg.Add(1)

		go func() {
			defer wg.Done()
			for p := range process {
				s.Suffix = fmt.Sprintf("%s ...\n", p)
			}
		}()
		s.Start()

		repos, err := getRepos(username, language, maxResults, process)
		if err != nil {
			fmt.Println("Error during repository fetch", err)
			close(process)
			os.Exit(0)
		}

		close(process)
		wg.Wait()
		s.Stop()

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
	reposCmd.Flags().StringP("language", "l", "all", "The language of the repository")
}

func getRepos(username, lang string, max int, process chan string) ([]repository, error) {
	process <- "Fetching data from github"
	url := fmt.Sprintf("https://api.github.com/users/%s/repos?per_page=%d&sort=update", username, max)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	process <- "Verifying response"
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch repositories for user %s: %s", username, resp.Status)
	}

	var repositories []repository
	if err := json.NewDecoder(resp.Body).Decode(&repositories); err != nil {
		return nil, err
	}

	var filteredRepos []repository
	process <- "Filtering repositories"
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

func renderRepos(repos []repository) (string, error) {
	headers := []string{"Name", "Description", "Url", "Last update"}
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
