package container

import (
	"bytes"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/docker/docker/api/types/container"
	apinetwork "github.com/docker/docker/api/types/network"
	"github.com/docker/go-sdk/container/exec"
	"github.com/docker/go-sdk/container/wait"
)

func TestWithAdditionalHostConfigModifier(t *testing.T) {
	t.Run("add-to-existing", func(t *testing.T) {
		def := Definition{
			image: "alpine",
			hostConfigModifier: func(hostConfig *container.HostConfig) {
				hostConfig.ContainerIDFile = "container-id-file"
			},
		}

		opt := WithAdditionalHostConfigModifier(func(hostConfig *container.HostConfig) {
			hostConfig.Binds = append(hostConfig.Binds, "/host/path:/container/path")
		})
		require.NoError(t, opt.Customize(&def))

		hc := container.HostConfig{}

		def.hostConfigModifier(&hc)
		require.Equal(t, "container-id-file", hc.ContainerIDFile)
		require.Equal(t, []string{"/host/path:/container/path"}, hc.Binds)
	})

	t.Run("nil-original", func(t *testing.T) {
		def := Definition{}

		opt := WithAdditionalHostConfigModifier(func(hostConfig *container.HostConfig) {
			hostConfig.Binds = append(hostConfig.Binds, "/host/path:/container/path")
		})
		require.NoError(t, opt.Customize(&def))
	})
}

func TestWithAdditionalConfigModifier(t *testing.T) {
	t.Run("add-to-existing", func(t *testing.T) {
		def := Definition{
			image: "alpine",
			configModifier: func(config *container.Config) {
				config.Env = append(config.Env, "ENV1=value1", "ENV2=value2")
				config.Hostname = "test-hostname-1"
			},
		}

		opt := WithAdditionalConfigModifier(func(config *container.Config) {
			config.Env = append(config.Env, "ENV3=value3", "ENV4=value4")
			config.Hostname = "test-hostname-2"
		})
		require.NoError(t, opt.Customize(&def))

		config := container.Config{}

		def.configModifier(&config)
		require.Equal(t, []string{"ENV1=value1", "ENV2=value2", "ENV3=value3", "ENV4=value4"}, config.Env)
		require.Equal(t, "test-hostname-2", config.Hostname)
	})

	t.Run("nil-original", func(t *testing.T) {
		def := Definition{}

		opt := WithAdditionalConfigModifier(func(config *container.Config) {
			config.Env = append(config.Env, "ENV3=value3", "ENV4=value4")
			config.Hostname = "test-hostname-2"
		})
		require.NoError(t, opt.Customize(&def))
	})
}

func TestWithAdditionalEndpointSettingsModifier(t *testing.T) {
	t.Run("add-to-existing", func(t *testing.T) {
		def := Definition{
			image: "alpine",
			endpointSettingsModifier: func(settings map[string]*apinetwork.EndpointSettings) {
				settings["test-network"] = &apinetwork.EndpointSettings{
					Aliases: []string{"alias1", "alias2"},
				}
			},
		}

		opt := WithAdditionalEndpointSettingsModifier(func(settings map[string]*apinetwork.EndpointSettings) {
			settings["test-network"] = &apinetwork.EndpointSettings{
				Links: []string{"link1:alias1", "link2:alias2"},
			}
		})
		require.NoError(t, opt.Customize(&def))

		endpointSettings := map[string]*apinetwork.EndpointSettings{}

		def.endpointSettingsModifier(endpointSettings)
		require.Contains(t, endpointSettings, "test-network")
		require.Equal(t, []string{"alias1", "alias2"}, endpointSettings["test-network"].Aliases)
		require.Equal(t, []string{"link1:alias1", "link2:alias2"}, endpointSettings["test-network"].Links)
	})

	t.Run("nil-original", func(t *testing.T) {
		def := Definition{}

		opt := WithAdditionalEndpointSettingsModifier(func(settings map[string]*apinetwork.EndpointSettings) {
			settings["test-network"] = &apinetwork.EndpointSettings{
				Aliases: []string{"alias1", "alias2"},
			}
		})
		require.NoError(t, opt.Customize(&def))
	})
}

func TestWithStartupCommand(t *testing.T) {
	def := Definition{
		image:      "alpine",
		entrypoint: []string{"tail", "-f", "/dev/null"},
		started:    true,
	}

	testExec := exec.NewRawCommand([]string{"touch", ".go-sdk"}, exec.WithWorkingDir("/tmp"))

	err := WithStartupCommand(testExec)(&def)
	require.NoError(t, err)

	require.Len(t, def.lifecycleHooks, 1)
	require.Len(t, def.lifecycleHooks[0].PostStarts, 1)
}

func TestWithAfterReadyCommand(t *testing.T) {
	def := Definition{
		image:      "alpine",
		entrypoint: []string{"tail", "-f", "/dev/null"},
		started:    true,
	}

	testExec := exec.NewRawCommand([]string{"touch", "/tmp/.go-sdk"})

	err := WithAfterReadyCommand(testExec)(&def)
	require.NoError(t, err)

	require.Len(t, def.lifecycleHooks, 1)
	require.Len(t, def.lifecycleHooks[0].PostReadies, 1)
}

func TestWithEnv(t *testing.T) {
	testEnv := func(t *testing.T, initial map[string]string, add map[string]string, expected map[string]string) {
		t.Helper()

		def := Definition{
			env: initial,
		}
		opt := WithEnv(add)
		require.NoError(t, opt.Customize(&def))
		require.Equal(t, expected, def.env)
	}

	t.Run("add-to-existing", func(t *testing.T) {
		testEnv(t,
			map[string]string{"KEY1": "VAL1"},
			map[string]string{"KEY2": "VAL2"},
			map[string]string{
				"KEY1": "VAL1",
				"KEY2": "VAL2",
			},
		)
	})

	t.Run("add-to-nil", func(t *testing.T) {
		testEnv(t,
			nil,
			map[string]string{"KEY2": "VAL2"},
			map[string]string{"KEY2": "VAL2"},
		)
	})

	t.Run("override-existing", func(t *testing.T) {
		testEnv(t,
			map[string]string{
				"KEY1": "VAL1",
				"KEY2": "VAL2",
			},
			map[string]string{"KEY2": "VAL3"},
			map[string]string{
				"KEY1": "VAL1",
				"KEY2": "VAL3",
			},
		)
	})
}

func TestWithEntrypoint(t *testing.T) {
	testEntrypoint := func(t *testing.T, initial []string, add []string, expected []string) {
		t.Helper()

		def := Definition{
			entrypoint: initial,
		}
		opt := WithEntrypoint(add...)
		require.NoError(t, opt.Customize(&def))
		require.Equal(t, expected, def.entrypoint)
	}

	t.Run("replace-existing", func(t *testing.T) {
		testEntrypoint(t,
			[]string{"/bin/sh"},
			[]string{"pwd"},
			[]string{"pwd"},
		)
	})

	t.Run("replace-nil", func(t *testing.T) {
		testEntrypoint(t,
			nil,
			[]string{"/bin/sh", "-c"},
			[]string{"/bin/sh", "-c"},
		)
	})
}

func TestWithEntrypointArgs(t *testing.T) {
	testEntrypoint := func(t *testing.T, initial []string, add []string, expected []string) {
		t.Helper()

		def := Definition{
			entrypoint: initial,
		}
		opt := WithEntrypointArgs(add...)
		require.NoError(t, opt.Customize(&def))
		require.Equal(t, expected, def.entrypoint)
	}

	t.Run("add-to-existing", func(t *testing.T) {
		testEntrypoint(t,
			[]string{"/bin/sh"},
			[]string{"-c", "echo hello"},
			[]string{"/bin/sh", "-c", "echo hello"},
		)
	})

	t.Run("add-to-nil", func(t *testing.T) {
		testEntrypoint(t,
			nil,
			[]string{"/bin/sh", "-c"},
			[]string{"/bin/sh", "-c"},
		)
	})
}

func TestWithExposedPorts(t *testing.T) {
	testPorts := func(t *testing.T, initial []string, add []string, expected []string) {
		t.Helper()

		def := Definition{
			exposedPorts: initial,
		}
		opt := WithExposedPorts(add...)
		require.NoError(t, opt.Customize(&def))
		require.Equal(t, expected, def.exposedPorts)
	}

	t.Run("add-to-existing", func(t *testing.T) {
		testPorts(t,
			[]string{"8080/tcp"},
			[]string{"9090/tcp"},
			[]string{"8080/tcp", "9090/tcp"},
		)
	})

	t.Run("add-to-nil", func(t *testing.T) {
		testPorts(t,
			nil,
			[]string{"8080/tcp"},
			[]string{"8080/tcp"},
		)
	})
}

func TestWithCmd(t *testing.T) {
	testCmd := func(t *testing.T, initial []string, add []string, expected []string) {
		t.Helper()

		def := Definition{
			cmd: initial,
		}
		opt := WithCmd(add...)
		require.NoError(t, opt.Customize(&def))
		require.Equal(t, expected, def.cmd)
	}

	t.Run("replace-existing", func(t *testing.T) {
		testCmd(t,
			[]string{"echo"},
			[]string{"hello", "world"},
			[]string{"hello", "world"},
		)
	})

	t.Run("replace-nil", func(t *testing.T) {
		testCmd(t,
			nil,
			[]string{"echo", "hello"},
			[]string{"echo", "hello"},
		)
	})
}

func TestWithAlwaysPull(t *testing.T) {
	def := Definition{
		image: "alpine",
	}

	opt := WithAlwaysPull()
	require.NoError(t, opt.Customize(&def))
	require.True(t, def.alwaysPullImage)
}

func TestWithImagePlatform(t *testing.T) {
	def := Definition{
		image: "alpine",
	}

	opt := WithImagePlatform("linux/amd64")
	require.NoError(t, opt.Customize(&def))
	require.Equal(t, "linux/amd64", def.imagePlatform)
}

func TestWithCmdArgs(t *testing.T) {
	testCmd := func(t *testing.T, initial []string, add []string, expected []string) {
		t.Helper()

		def := Definition{
			cmd: initial,
		}
		opt := WithCmdArgs(add...)
		require.NoError(t, opt.Customize(&def))
		require.Equal(t, expected, def.cmd)
	}

	t.Run("add-to-existing", func(t *testing.T) {
		testCmd(t,
			[]string{"echo"},
			[]string{"hello", "world"},
			[]string{"echo", "hello", "world"},
		)
	})

	t.Run("add-to-nil", func(t *testing.T) {
		testCmd(t,
			nil,
			[]string{"echo", "hello"},
			[]string{"echo", "hello"},
		)
	})
}

func TestWithLabels(t *testing.T) {
	testLabels := func(t *testing.T, initial map[string]string, add map[string]string, expected map[string]string) {
		t.Helper()

		def := Definition{
			labels: initial,
		}
		opt := WithLabels(add)
		require.NoError(t, opt.Customize(&def))
		require.Equal(t, expected, def.labels)
	}

	t.Run("add-to-existing", func(t *testing.T) {
		testLabels(t,
			map[string]string{"key1": "value1"},
			map[string]string{"key2": "value2"},
			map[string]string{"key1": "value1", "key2": "value2"},
		)
	})

	t.Run("add-to-nil", func(t *testing.T) {
		testLabels(t,
			nil,
			map[string]string{"key1": "value1"},
			map[string]string{"key1": "value1"},
		)
	})
}

func TestWithLifecycleHooks(t *testing.T) {
	testHook := DefaultLoggingHook

	testLifecycleHooks := func(t *testing.T, replace bool, initial []LifecycleHooks, add []LifecycleHooks, expected []LifecycleHooks) {
		t.Helper()

		def := Definition{
			lifecycleHooks: initial,
		}

		var opt CustomizeDefinitionOption
		if replace {
			opt = WithLifecycleHooks(add...)
		} else {
			opt = WithAdditionalLifecycleHooks(add...)
		}
		require.NoError(t, opt.Customize(&def))
		require.Len(t, def.lifecycleHooks, len(expected))
		for i, hook := range expected {
			require.Equal(t, hook, def.lifecycleHooks[i])
		}
	}

	t.Run("replace-nil", func(t *testing.T) {
		testLifecycleHooks(t,
			true,
			nil,
			[]LifecycleHooks{testHook},
			[]LifecycleHooks{testHook},
		)
	})

	t.Run("replace-existing", func(t *testing.T) {
		testLifecycleHooks(t,
			true,
			[]LifecycleHooks{testHook},
			[]LifecycleHooks{testHook},
			[]LifecycleHooks{testHook},
		)
	})

	t.Run("add-to-nil", func(t *testing.T) {
		testLifecycleHooks(t,
			false,
			nil,
			[]LifecycleHooks{testHook},
			[]LifecycleHooks{testHook},
		)
	})

	t.Run("add-to-existing", func(t *testing.T) {
		testLifecycleHooks(t,
			false,
			[]LifecycleHooks{testHook},
			[]LifecycleHooks{testHook},
			[]LifecycleHooks{testHook, testHook},
		)
	})
}

func TestWithFiles(t *testing.T) {
	testFiles := func(t *testing.T, initial []File, add []File, expected []File) {
		t.Helper()

		def := Definition{
			files: initial,
		}
		opt := WithFiles(add...)
		require.NoError(t, opt.Customize(&def))
		require.Equal(t, expected, def.files)
	}

	reader1 := bytes.NewReader([]byte("hello"))
	reader2 := bytes.NewReader([]byte("world"))

	t.Run("add-to-existing", func(t *testing.T) {
		testFiles(t,
			[]File{{Reader: reader1, ContainerPath: "/container/file1"}},
			[]File{{Reader: reader2, ContainerPath: "/container/file2"}},
			[]File{
				{Reader: reader1, ContainerPath: "/container/file1"},
				{Reader: reader2, ContainerPath: "/container/file2"},
			},
		)
	})

	t.Run("add-to-nil", func(t *testing.T) {
		testFiles(t,
			nil,
			[]File{{Reader: reader1, ContainerPath: "/container/file1"}},
			[]File{{Reader: reader1, ContainerPath: "/container/file1"}},
		)
	})
}

func TestWithName(t *testing.T) {
	def := Definition{}

	opt := WithName("pg-test")
	require.NoError(t, opt.Customize(&def))
	require.Equal(t, "pg-test", def.name)

	t.Run("empty", func(t *testing.T) {
		def := Definition{}

		opt := WithName("")
		require.ErrorIs(t, opt.Customize(&def), ErrReuseEmptyName)
	})
}

func TestWithNoStart(t *testing.T) {
	def := Definition{}

	opt := WithNoStart()
	require.NoError(t, opt.Customize(&def))
	require.False(t, def.started)
}

func TestWithWaitStrategy(t *testing.T) {
	testDuration := 10 * time.Second
	defaultDuration := 60 * time.Second

	waitForFoo := wait.ForLog("foo")
	waitForBar := wait.ForLog("bar")

	testWaitFor := func(t *testing.T, replace bool, customDuration *time.Duration, initial wait.Strategy, add wait.Strategy, expected wait.Strategy) {
		t.Helper()

		def := Definition{
			waitingFor: initial,
		}

		var opt CustomizeDefinitionOption
		if replace {
			opt = WithWaitStrategy(add)
			if customDuration != nil {
				opt = WithWaitStrategyAndDeadline(*customDuration, add)
			}
		} else {
			opt = WithAdditionalWaitStrategy(add)
			if customDuration != nil {
				opt = WithAdditionalWaitStrategyAndDeadline(*customDuration, add)
			}
		}
		require.NoError(t, opt.Customize(&def))
		require.Equal(t, expected, def.waitingFor)
	}

	t.Run("replace-nil", func(t *testing.T) {
		t.Run("default-duration", func(t *testing.T) {
			testWaitFor(t,
				true,
				nil,
				nil,
				waitForFoo,
				wait.ForAll(waitForFoo).WithDeadline(defaultDuration),
			)
		})

		t.Run("custom-duration", func(t *testing.T) {
			testWaitFor(t,
				true,
				&testDuration,
				nil,
				waitForFoo,
				wait.ForAll(waitForFoo).WithDeadline(testDuration),
			)
		})
	})

	t.Run("replace-existing", func(t *testing.T) {
		t.Run("default-duration", func(t *testing.T) {
			testWaitFor(t,
				true,
				nil,
				waitForFoo,
				waitForBar,
				wait.ForAll(waitForBar).WithDeadline(defaultDuration),
			)
		})

		t.Run("custom-duration", func(t *testing.T) {
			testWaitFor(t,
				true,
				&testDuration,
				waitForFoo,
				waitForBar,
				wait.ForAll(waitForBar).WithDeadline(testDuration),
			)
		})
	})

	t.Run("add-to-nil", func(t *testing.T) {
		t.Run("default-duration", func(t *testing.T) {
			testWaitFor(t,
				false,
				nil,
				nil,
				waitForFoo,
				wait.ForAll(waitForFoo).WithDeadline(defaultDuration),
			)
		})

		t.Run("custom-duration", func(t *testing.T) {
			testWaitFor(t,
				false,
				&testDuration,
				nil,
				waitForFoo,
				wait.ForAll(waitForFoo).WithDeadline(testDuration),
			)
		})
	})

	t.Run("add-to-existing", func(t *testing.T) {
		t.Run("default-duration", func(t *testing.T) {
			testWaitFor(t,
				false,
				nil,
				waitForFoo,
				waitForBar,
				wait.ForAll(waitForFoo, waitForBar).WithDeadline(defaultDuration),
			)
		})

		t.Run("custom-duration", func(t *testing.T) {
			testWaitFor(t,
				false,
				&testDuration,
				waitForFoo,
				waitForBar,
				wait.ForAll(waitForFoo, waitForBar).WithDeadline(testDuration),
			)
		})
	})
}

func TestWithValidateFuncs(t *testing.T) {
	t.Run("add-zero", func(t *testing.T) {
		def := Definition{}

		opt := WithValidateFuncs()
		require.ErrorContains(t, opt.Customize(&def), "validate function is nil")
		require.Empty(t, def.validateFuncs)
	})

	t.Run("add-nil", func(t *testing.T) {
		def := Definition{}

		opt := WithValidateFuncs(nil)
		require.ErrorContains(t, opt.Customize(&def), "validate function is nil")
		require.Empty(t, def.validateFuncs)
	})

	t.Run("add-one-nil", func(t *testing.T) {
		def := Definition{}

		opt := WithValidateFuncs(func() error {
			return nil
		}, nil)
		require.ErrorContains(t, opt.Customize(&def), "validate function is nil")
		require.Empty(t, def.validateFuncs)
	})

	t.Run("add-single", func(t *testing.T) {
		def := Definition{}

		opt := WithValidateFuncs(func() error {
			return errors.New("test error")
		})
		require.NoError(t, opt.Customize(&def))
		require.Len(t, def.validateFuncs, 1)
	})

	t.Run("add-multiple", func(t *testing.T) {
		def := Definition{}

		opt := WithValidateFuncs(
			func() error {
				return errors.New("test error")
			},
			func() error {
				return errors.New("test error 2")
			},
		)
		require.NoError(t, opt.Customize(&def))
		require.Len(t, def.validateFuncs, 2)
	})
}

func TestWithDefinition(t *testing.T) {
	def1 := Definition{
		image: "alpine",
	}

	def2 := Definition{
		image: "busybox",
	}

	opt := WithDefinition(&def2)
	require.NoError(t, opt.Customize(&def1))
	require.Equal(t, "alpine", def1.image)
	require.Equal(t, "alpine", def2.image)
}

func TestWithPullHandler(t *testing.T) {
	def := Definition{}

	opt := WithPullHandler(func(_ io.ReadCloser) error {
		return nil
	})
	require.NoError(t, opt.Customize(&def))
	require.Len(t, def.pullOptions, 1)
}
