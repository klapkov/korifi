package presenter

type V2InfoResponse struct {
	Name                     string `json:"name"`
	Build                    string `json:"build"`
	Support                  string `json:"support"`
	Version                  int    `json:"version"`
	Description              string `json:"description"`
	AuthEndpoint             string `json:"authorization_endpoint"`
	TokenEndpoint            string `json:"token_endpoint"`
	MinCLIVersion            string `json:"min_cli_version"`
	MinRecommendedCLIVersion string `json:"min_recommended_cli_version"`
	AppSSHEndpoint           string `json:"app_ssh_endpoint"`
	HostFingerprint          string `json:"app_ssh_host_key_fingerprint"`
	SSHOauthClient           string `json:"app_ssh_oauth_client"`
	DopplerLoggingEndpoint   string `json:"doppler_logging_endpoint"`
	ApiVersion               string `json:"api_version"`
	OsbapiVersion            string `json:"osbapi_version"`
	User                     string `json:"user"`
}

func ForV2Info() V2InfoResponse {
	return V2InfoResponse{
		Name:                     "cf-deployment",
		Build:                    "v48.9.0",
		Support:                  "",
		Version:                  48,
		Description:              "SAP BTP Cloud Foundry environment",
		AuthEndpoint:             "https://login.cf.korifi-dev.cfrt-sof.sapcloud.io",
		TokenEndpoint:            "https://uaa.cf.korifi-dev.cfrt-sof.sapcloud.io",
		MinCLIVersion:            "8.0.0",
		MinRecommendedCLIVersion: "",
		AppSSHEndpoint:           "https://ssh.cf.korifi-dev.cfrt-sof.sapcloud.io",
		HostFingerprint:          "4Uzl0f+vveletSDGTKLXFiVo8hEpBa7Luzb2l1rpuMU",
		SSHOauthClient:           "ssh-proxy",
		DopplerLoggingEndpoint:   "https://doppler.cf.korifi-dev.cfrt-sof.sapcloud.io",
		ApiVersion:               "2.256.0",
		OsbapiVersion:            "2.15",
		User:                     "some-user",
	}
}
