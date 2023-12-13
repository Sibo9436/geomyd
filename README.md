# Geomyd
A web-fetcher and some 
## How to 
The easiest way to get it up and running is running
```bash
cd geomyd
docker build -t geomyd .
alias geomyd='docker run --rm -v .:/usr/src/geomyd:Z geomyd'
geomyd --help
```

## Functionalities
Geomyd will download any webpage for you, optionally providing you with metadata about the fetch or even downloading image assets for offline browsing

## Known issues
Due to how the directory structure/link referencing works, some websites that reference their asset with a leading / will not display the image correctly while browsing offline, a current workaround is opening the devtools and removing said leading slash 
(Known examples: https://www.google.com breaks - https://wikipedia.org works correctly)

