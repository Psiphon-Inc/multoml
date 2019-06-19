This library is **DEPRECATED** in favour of https://github.com/Psiphon-Inc/configloader-go

# multoml

multoml merges multiple TOML files into one structure. It is intended to be used for configuration loading, where the config is held in a (source-controlled) main file and a (local-only) override file.

multoml also provides the ability to override file values from environment variables.

## Usage

See the [GoDoc](https://godoc.org/github.com/Psiphon-Inc/multoml) for more info.

```no-highlight
go get github.com/Psiphon-Inc/multoml
```

```golang
import multoml

// The first file must exist in one of the search paths, but the subsequent ones needn't.
filenames := []string{"config.toml", "config_override.toml"}

// Places where we'll look for the config files, in order.
searchPaths := []string{".", "/etc/config"}

// Values from the files can be overridden by env vars, if they're set.
envOverrides := map[string]string{"DATABASE_HOST": "database.host"}

conf := multoml.NewFromFiles(filenames, searchPaths, envOverrides)
fmt.Println(conf.Get("database.host").(string))

// Alternatively...
readers := []io.Reader{...}
conf = multoml.NewFromReaders(readers, envOverrides)
```

## License

BSD 3-Clause License.
