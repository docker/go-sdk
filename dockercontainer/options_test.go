package dockercontainer_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/docker/go-sdk/dockercontainer"
	"github.com/docker/go-sdk/dockercontainer/exec"
	"github.com/docker/go-sdk/dockercontainer/wait"
)

type msgsLogConsumer struct {
	msgs []string
}

// Accept prints the log to stdout
func (lc *msgsLogConsumer) Accept(l dockercontainer.Log) {
	lc.msgs = append(lc.msgs, string(l.Content))
}

func TestWithLogConsumers(t *testing.T) {
	lc := &msgsLogConsumer{}
	def := dockercontainer.Definition{
		Image:      "mysql:8.0.36",
		WaitingFor: wait.ForLog("port: 3306  MySQL Community Server - GPL"),
		Started:    true,
	}

	err := dockercontainer.WithLogConsumers(lc)(&def)
	require.NoError(t, err)
}

func TestWithLogConsumerConfig(t *testing.T) {
	lc := &msgsLogConsumer{}

	t.Run("add-to-nil", func(t *testing.T) {
		def := dockercontainer.Definition{
			Image: "alpine",
		}

		err := dockercontainer.WithLogConsumerConfig(&dockercontainer.LogConsumerConfig{
			Consumers: []dockercontainer.LogConsumer{lc},
		})(&def)
		require.NoError(t, err)

		require.Equal(t, []dockercontainer.LogConsumer{lc}, def.LogConsumerCfg.Consumers)
	})

	t.Run("replace-existing", func(t *testing.T) {
		def := dockercontainer.Definition{
			Image: "alpine",
			LogConsumerCfg: &dockercontainer.LogConsumerConfig{
				Consumers: []dockercontainer.LogConsumer{dockercontainer.NewFooLogConsumer(t)},
			},
		}

		err := dockercontainer.WithLogConsumerConfig(&dockercontainer.LogConsumerConfig{
			Consumers: []dockercontainer.LogConsumer{lc},
		})(&def)
		require.NoError(t, err)

		require.Equal(t, []dockercontainer.LogConsumer{lc}, def.LogConsumerCfg.Consumers)
	})
}

func TestWithStartupCommand(t *testing.T) {
	def := dockercontainer.Definition{
		Image:      "alpine",
		Entrypoint: []string{"tail", "-f", "/dev/null"},
		Started:    true,
	}

	testExec := exec.NewRawCommand([]string{"touch", ".testcontainers"}, exec.WithWorkingDir("/tmp"))

	err := dockercontainer.WithStartupCommand(testExec)(&def)
	require.NoError(t, err)

	require.Len(t, def.LifecycleHooks, 1)
	require.Len(t, def.LifecycleHooks[0].PostStarts, 1)
}

func TestWithAfterReadyCommand(t *testing.T) {
	def := dockercontainer.Definition{
		Image:      "alpine",
		Entrypoint: []string{"tail", "-f", "/dev/null"},
		Started:    true,
	}

	testExec := exec.NewRawCommand([]string{"touch", "/tmp/.testcontainers"})

	err := dockercontainer.WithAfterReadyCommand(testExec)(&def)
	require.NoError(t, err)

	require.Len(t, def.LifecycleHooks, 1)
	require.Len(t, def.LifecycleHooks[0].PostReadies, 1)
}

func TestWithEnv(t *testing.T) {
	testEnv := func(t *testing.T, initial map[string]string, add map[string]string, expected map[string]string) {
		t.Helper()

		def := dockercontainer.Definition{
			Env: initial,
		}
		opt := dockercontainer.WithEnv(add)
		require.NoError(t, opt.Customize(&def))
		require.Equal(t, expected, def.Env)
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

		def := dockercontainer.Definition{
			Entrypoint: initial,
		}
		opt := dockercontainer.WithEntrypoint(add...)
		require.NoError(t, opt.Customize(&def))
		require.Equal(t, expected, def.Entrypoint)
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

		def := dockercontainer.Definition{
			Entrypoint: initial,
		}
		opt := dockercontainer.WithEntrypointArgs(add...)
		require.NoError(t, opt.Customize(&def))
		require.Equal(t, expected, def.Entrypoint)
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

		def := dockercontainer.Definition{
			ExposedPorts: initial,
		}
		opt := dockercontainer.WithExposedPorts(add...)
		require.NoError(t, opt.Customize(&def))
		require.Equal(t, expected, def.ExposedPorts)
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

		def := dockercontainer.Definition{
			Cmd: initial,
		}
		opt := dockercontainer.WithCmd(add...)
		require.NoError(t, opt.Customize(&def))
		require.Equal(t, expected, def.Cmd)
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
	def := dockercontainer.Definition{
		Image: "alpine",
	}

	opt := dockercontainer.WithAlwaysPull()
	require.NoError(t, opt.Customize(&def))
	require.True(t, def.AlwaysPullImage)
}

func TestWithImagePlatform(t *testing.T) {
	def := dockercontainer.Definition{
		Image: "alpine",
	}

	opt := dockercontainer.WithImagePlatform("linux/amd64")
	require.NoError(t, opt.Customize(&def))
	require.Equal(t, "linux/amd64", def.ImagePlatform)
}

func TestWithCmdArgs(t *testing.T) {
	testCmd := func(t *testing.T, initial []string, add []string, expected []string) {
		t.Helper()

		def := dockercontainer.Definition{
			Cmd: initial,
		}
		opt := dockercontainer.WithCmdArgs(add...)
		require.NoError(t, opt.Customize(&def))
		require.Equal(t, expected, def.Cmd)
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

		def := dockercontainer.Definition{
			Labels: initial,
		}
		opt := dockercontainer.WithLabels(add)
		require.NoError(t, opt.Customize(&def))
		require.Equal(t, expected, def.Labels)
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
	testHook := dockercontainer.DefaultLoggingHook(nil)

	testLifecycleHooks := func(t *testing.T, replace bool, initial []dockercontainer.LifecycleHooks, add []dockercontainer.LifecycleHooks, expected []dockercontainer.LifecycleHooks) {
		t.Helper()

		def := dockercontainer.Definition{
			LifecycleHooks: initial,
		}

		var opt dockercontainer.CustomizeDefinitionOption
		if replace {
			opt = dockercontainer.WithLifecycleHooks(add...)
		} else {
			opt = dockercontainer.WithAdditionalLifecycleHooks(add...)
		}
		require.NoError(t, opt.Customize(&def))
		require.Len(t, def.LifecycleHooks, len(expected))
		for i, hook := range expected {
			require.Equal(t, hook, def.LifecycleHooks[i])
		}
	}

	t.Run("replace-nil", func(t *testing.T) {
		testLifecycleHooks(t,
			true,
			nil,
			[]dockercontainer.LifecycleHooks{testHook},
			[]dockercontainer.LifecycleHooks{testHook},
		)
	})

	t.Run("replace-existing", func(t *testing.T) {
		testLifecycleHooks(t,
			true,
			[]dockercontainer.LifecycleHooks{testHook},
			[]dockercontainer.LifecycleHooks{testHook},
			[]dockercontainer.LifecycleHooks{testHook},
		)
	})

	t.Run("add-to-nil", func(t *testing.T) {
		testLifecycleHooks(t,
			false,
			nil,
			[]dockercontainer.LifecycleHooks{testHook},
			[]dockercontainer.LifecycleHooks{testHook},
		)
	})

	t.Run("add-to-existing", func(t *testing.T) {
		testLifecycleHooks(t,
			false,
			[]dockercontainer.LifecycleHooks{testHook},
			[]dockercontainer.LifecycleHooks{testHook},
			[]dockercontainer.LifecycleHooks{testHook, testHook},
		)
	})
}

func TestWithFiles(t *testing.T) {
	testFiles := func(t *testing.T, initial []dockercontainer.File, add []dockercontainer.File, expected []dockercontainer.File) {
		t.Helper()

		def := dockercontainer.Definition{
			Files: initial,
		}
		opt := dockercontainer.WithFiles(add...)
		require.NoError(t, opt.Customize(&def))
		require.Equal(t, expected, def.Files)
	}

	reader1 := bytes.NewReader([]byte("hello"))
	reader2 := bytes.NewReader([]byte("world"))

	t.Run("add-to-existing", func(t *testing.T) {
		testFiles(t,
			[]dockercontainer.File{{Reader: reader1, ContainerPath: "/container/file1"}},
			[]dockercontainer.File{{Reader: reader2, ContainerPath: "/container/file2"}},
			[]dockercontainer.File{
				{Reader: reader1, ContainerPath: "/container/file1"},
				{Reader: reader2, ContainerPath: "/container/file2"},
			},
		)
	})

	t.Run("add-to-nil", func(t *testing.T) {
		testFiles(t,
			nil,
			[]dockercontainer.File{{Reader: reader1, ContainerPath: "/container/file1"}},
			[]dockercontainer.File{{Reader: reader1, ContainerPath: "/container/file1"}},
		)
	})
}

func TestWithName(t *testing.T) {
	t.Parallel()
	def := dockercontainer.Definition{}

	opt := dockercontainer.WithName("pg-test")
	require.NoError(t, opt.Customize(&def))
	require.Equal(t, "pg-test", def.Name)

	t.Run("empty", func(t *testing.T) {
		def := dockercontainer.Definition{}

		opt := dockercontainer.WithName("")
		require.ErrorIs(t, opt.Customize(&def), dockercontainer.ErrReuseEmptyName)
	})
}

func TestWithNoStart(t *testing.T) {
	t.Parallel()
	def := dockercontainer.Definition{}

	opt := dockercontainer.WithNoStart()
	require.NoError(t, opt.Customize(&def))
	require.False(t, def.Started)
}

func TestWithWaitStrategy(t *testing.T) {
	testDuration := 10 * time.Second
	defaultDuration := 60 * time.Second

	waitForFoo := wait.ForLog("foo")
	waitForBar := wait.ForLog("bar")

	testWaitFor := func(t *testing.T, replace bool, customDuration *time.Duration, initial wait.Strategy, add wait.Strategy, expected wait.Strategy) {
		t.Helper()

		def := dockercontainer.Definition{
			WaitingFor: initial,
		}

		var opt dockercontainer.CustomizeDefinitionOption
		if replace {
			opt = dockercontainer.WithWaitStrategy(add)
			if customDuration != nil {
				opt = dockercontainer.WithWaitStrategyAndDeadline(*customDuration, add)
			}
		} else {
			opt = dockercontainer.WithAdditionalWaitStrategy(add)
			if customDuration != nil {
				opt = dockercontainer.WithAdditionalWaitStrategyAndDeadline(*customDuration, add)
			}
		}
		require.NoError(t, opt.Customize(&def))
		require.Equal(t, expected, def.WaitingFor)
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
