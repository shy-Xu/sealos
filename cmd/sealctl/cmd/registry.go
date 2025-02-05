/*
Copyright 2022 cuisongliu@qq.com.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	"context"
	"os"
	"strings"

	"github.com/docker/docker/api/types"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/cobra"

	"github.com/labring/sealos/pkg/buildimage"
	"github.com/labring/sealos/pkg/passwd"
	"github.com/labring/sealos/pkg/registry"
	"github.com/labring/sealos/pkg/utils/file"
	"github.com/labring/sealos/pkg/utils/flags"
	"github.com/labring/sealos/pkg/utils/logger"
	"github.com/labring/sealos/pkg/utils/maps"
)

var (
	registryPullRegistryDir  string
	registryPullAuths        []string
	registryPullArch         string
	registryPullMaxPullProcs int
	registryPullAuthBasic    bool
)

func newRegistryImageCmd() *cobra.Command {
	var registryImageCmd = &cobra.Command{
		Use:   "registry",
		Short: "registry images manager",
		//Run: func(cmd *cobra.Command, args []string) {
		//
		//},
	}
	registryImageCmd.AddCommand(newRegistryImagePullCmd())
	return registryImageCmd
}

func newRegistryImagePullCmd() *cobra.Command {
	var registryImagePullCmd = &cobra.Command{
		Use:   "pull",
		Short: "registry images manager pull to local dir",
		//Run: func(cmd *cobra.Command, args []string) {
		//
		//},
	}
	registryImagePullCmd.PersistentFlags().StringVar(&registryPullArch, "arch", "amd64", "pull images arch")
	registryImagePullCmd.PersistentFlags().StringVar(&registryPullRegistryDir, "data-dir", "/var/lib/registry", "registry data dir path")
	registryImagePullCmd.PersistentFlags().StringSliceVar(&registryPullAuths, "auths", []string{}, "auths data for login mirror registry, format example is \"address=docker.io&&auth=YWRtaW46YWRtaW4=\".")
	registryImagePullCmd.PersistentFlags().IntVar(&registryPullMaxPullProcs, "max-pull-procs", 5, "maximum number of goroutines for pulling")
	registryImagePullCmd.PersistentFlags().BoolVar(&registryPullAuthBasic, "basic-auth", false, "pull image auth policy,default is token auth")
	registryImagePullCmd.AddCommand(newRegistryImagePullRawCmd())
	registryImagePullCmd.AddCommand(newRegistryImagePullYamlCmd())
	registryImagePullCmd.AddCommand(newRegistryImagePullDefaultCmd())
	return registryImagePullCmd
}

func newRegistryImagePullRawCmd() *cobra.Command {
	var imageFile string
	var auth map[string]types.AuthConfig
	var registryImagePullRaw = &cobra.Command{
		Use:   "raw",
		Short: "registry images manager pull to local dir by raw type",
		Run: func(cmd *cobra.Command, args []string) {
			flags.PrintFlags(cmd.Flags())
			images, err := file.ReadLines(imageFile)
			if err != nil {
				logger.Error("ImageFile convert images is error: %s", err.Error())
				os.Exit(1)
			}
			is := registry.NewImageSaver(context.Background(), registryPullMaxPullProcs, auth, registryPullAuthBasic)
			_, err = is.SaveImages(images, registryPullRegistryDir, v1.Platform{OS: "linux", Architecture: registryPullArch})
			if err != nil {
				logger.Error("pull registry images is error: %s", err.Error())
				os.Exit(1)
			}
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			auth = validateRegistryImagePull()
			if !file.IsExist(imageFile) {
				logger.Error("ImageFile path is not exist")
				os.Exit(1)
			}
		},
	}

	registryImagePullRaw.PersistentFlags().StringVarP(&imageFile, "image-file", "f", "ImageFile", "ImageFile path")
	return registryImagePullRaw
}

func newRegistryImagePullYamlCmd() *cobra.Command {
	var yamlPath string
	var auth map[string]types.AuthConfig
	var registryImagePullYaml = &cobra.Command{
		Use:   "yaml",
		Short: "registry images manager pull to local dir by yaml type",
		Run: func(cmd *cobra.Command, args []string) {
			flags.PrintFlags(cmd.Flags())
			images, err := buildimage.ParseYamlImages(yamlPath)
			if err != nil {
				logger.Error("yaml path convert images is error: %s", err.Error())
				os.Exit(1)
			}
			is := registry.NewImageSaver(context.Background(), registryPullMaxPullProcs, auth, registryPullAuthBasic)
			_, err = is.SaveImages(images, registryPullRegistryDir, v1.Platform{OS: "linux", Architecture: registryPullArch})
			if err != nil {
				logger.Error("pull registry images is error: %s", err.Error())
				os.Exit(1)
			}
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			auth = validateRegistryImagePull()
			if !file.IsExist(yamlPath) {
				logger.Error("yaml data dir path is not exist")
				os.Exit(1)
			}
		},
	}

	registryImagePullYaml.PersistentFlags().StringVarP(&yamlPath, "yaml-path", "p", "", "yaml data dir path")
	return registryImagePullYaml
}

func newRegistryImagePullDefaultCmd() *cobra.Command {
	var images []string
	var auth map[string]types.AuthConfig
	var registryImagePullDefault = &cobra.Command{
		Use:   "default",
		Short: "registry images manager pull to local dir by default type",
		Run: func(cmd *cobra.Command, args []string) {
			flags.PrintFlags(cmd.Flags())
			is := registry.NewImageSaver(context.Background(), registryPullMaxPullProcs, auth, registryPullAuthBasic)
			_, err := is.SaveImages(images, registryPullRegistryDir, v1.Platform{OS: "linux", Architecture: registryPullArch})
			if err != nil {
				logger.Error("pull registry images is error: %s", err.Error())
				os.Exit(1)
			}
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			auth = validateRegistryImagePull()
		},
	}

	registryImagePullDefault.PersistentFlags().StringSliceVar(&images, "images", []string{}, "images list")
	return registryImagePullDefault
}

func validateRegistryImagePull() map[string]types.AuthConfig {
	if !file.IsExist(registryPullRegistryDir) {
		logger.Error("registry data dir is not exist")
		os.Exit(1)
	}
	data := make(map[string]types.AuthConfig)
	for _, a := range registryPullAuths {
		auth := maps.StringToMap(a, "&&")
		var ok bool
		logger.Debug("range auth: %v", auth)
		if _, ok = auth["address"]; !ok {
			logger.Error("auths format is error, format is \"address=docker.io&&auth=YWRtaW46YWRtaW4=\".")
			os.Exit(1)
		} else {
			userAndPwd, _ := passwd.LoginAuthDecode(auth["auth"])
			authConfig := types.AuthConfig{
				ServerAddress: auth["address"],
			}
			if userAndPwd != "" {
				if userAndPwdArr := strings.Split(userAndPwd, ":"); len(userAndPwdArr) == 2 {
					authConfig.Username = userAndPwdArr[0]
					authConfig.Password = userAndPwdArr[1]
				}
			}
			data[authConfig.ServerAddress] = authConfig
		}
	}
	return data
}

func init() {
	rootCmd.AddCommand(newRegistryImageCmd())
}
