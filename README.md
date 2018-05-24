# transmission-filter

## Overview

This program look at transmission list of finished torrent and move those to
matching folder name inside a provided out directory.

## Usage

```shell
transmission-filter [OPTIONS]

Application Options:
  -o, --out=       Output directory (default: .)
  -u, --url=       Transmission url (default: 127.0.0.1:9091)
  -e, --extension= File extension to filter (default: mp4,mkv,avi,srt,mp3,ogg)

Help Options:
  -h, --help       Show this help message
```

## License

Copyright (C) 2018 Aurélien Chabot <aurelien@chabot.fr>

Licensed under the **MIT License**
