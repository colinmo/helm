package main

var JiraFixtures = "/Users/s457972/Dropbox/swap/golang/helm/fixtures/"

/*
func TestConvertFileToData(t *testing.T) {
	j := Jira{}
	bob := MarkdownTask{}
	bob = j.convertFileToData(path.Join(JiraFixtures, "local-jira/has-jira/20220101-Title.md"))
	if bob.JiraID != "SA-134" {
		t.Fatalf("Didn't get JiraID %s\n", bob.JiraID)
	}
	if bob.Body == "" {
		t.Fatalf("No body to love")
	}

	bob = j.convertFileToData(path.Join(JiraFixtures, "local-jira/no-jira/20220101-Title.md"))
	if bob.JiraID != "" {
		t.Fatalf("Where'd you come from %s\n", bob.JiraID)
	}
	if bob.Body == "" {
		t.Fatalf("No body to love")
	}
}

func TestCompareLocalToRemoteSingle(t *testing.T) {
	j := Jira{}
	bob := j.CompareLocalToRemoteSingle("Me", "a\nb\nv", time.Now().Format(JiraDateTimeStampFormat), "a\nc\nv", time.Now().Format(JiraDateTimeStampFormat))
	if bob.LocalFile != "Me" {
		t.Fatalf("Wrong local name")
	}
	if bob.RemoteFile != "Remote" {
		t.Fatalf("Wrong remote name")
	}
	if bob.Differences != "--- Me\n+++ Remote\n@@ -1,3 +1,3 @@\n a\n-b\n+c\n v\n\\ No newline at end of file\n" {
		t.Fatalf("Wrong differences '%s'", bob.Differences)
	}
}

func TestDifferenceObjects(t *testing.T) {
	j := Jira{}
	localMarkdowns := map[string]MarkdownTask{
		"SA-1": {
			JiraID:   "SA-1",
			Filename: "20220101-sa1.md",
			Body:     ".h1{mep}\n\ndude1",
			Comments: []string{},
			Metadata: JiraMetadata{
				Name: "First",
			},
		},
		"SA-2": {
			JiraID:   "SA-2",
			Filename: "20220101-sa2.md",
			Body:     ".h1{mep}\n\ndude2",
			Comments: []string{},
			Metadata: JiraMetadata{
				Name: "Second",
			},
		},
		"SA-4": {
			JiraID:   "SA-4",
			Filename: "20220101-sa4.md",
			Body:     ".h1{mep}\n\ndude",
			Comments: []string{},
			Metadata: JiraMetadata{
				Name: "Forth",
			},
		},
	}
	newMarkdowns := []MarkdownTask{
		{JiraID: "", Filename: "20230101-sa3.md", Body: ".h1{dddd}\n\nasdf\nasdf\nasdf", Metadata: JiraMetadata{Name: "Third"}},
	}
	remoteMarkdowns := map[string]MarkdownTask{
		"SA-1": {
			JiraID:   "SA-1",
			Filename: "20220101-sa1.md",
			Body:     ".h1{mep}\n\ndude1",
			Comments: []string{},
			Metadata: JiraMetadata{
				Name: "First",
			},
		},
		"SA-2": {
			JiraID:   "SA-2",
			Filename: "20220101-sa2.md",
			Body:     ".h1{mep}\n\ndude2",
			Comments: []string{},
			Metadata: JiraMetadata{
				Name: "Second",
			},
		},
		"SA-4": {
			JiraID:   "SA-4",
			Filename: "",
			Body:     ".h1{62356}\n\ndude",
			Comments: []string{},
			Metadata: JiraMetadata{
				Name: "Forth",
			},
		},
		"SA-5": {
			JiraID:   "SA-5",
			Filename: "",
			Body:     ".h1{234234}\n\ndude",
			Comments: []string{},
			Metadata: JiraMetadata{
				Name: "Fifth",
			},
		}}
	diffs := j.DifferenceObjects(&localMarkdowns, &newMarkdowns, &remoteMarkdowns)

	if len(diffs) != 3 {
		t.Fatalf("Wrong number of differences found: %d vs 3\n", len(diffs))
	}

	expected := JiraDifferences{
		LocalFile:   "20230101-sa3.md",
		RemoteFile:  "",
		Differences: "New Local File",
	}
	if diffs[0] != expected {
		t.Fatalf("Didn't figure the local file")
	}

	expected = JiraDifferences{
		LocalFile:   "20220101-sa4.md",
		RemoteFile:  "Remote",
		Differences: "--- 20220101-sa4.md\n+++ Remote\n@@ -1,3 +1,3 @@\n-.h1{mep}\n+.h1{62356}\n \n dude\n\\ No newline at end of file\n",
	}
	if diffs[1] != expected {
		t.Fatalf("Didn't figure the both file")
	}
	expected = JiraDifferences{
		LocalFile:   "",
		RemoteFile:  "Remote SA-5",
		Differences: "New Remote File",
	}
	if diffs[2] != expected {
		t.Fatalf("Didn't figure the remote file")
	}
}
*/
