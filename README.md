# ImportGoblin

ImportGoblin is a quick and dirty attempt to make my life with importing media files from various places info shared storage abreeze.

## Goals

- _minimal dependencies_ - written in golang, sqlite as base storage for now. Should allow compiling for other environments then linux-amd64 at some point as well.
- _directory and file naming_ - unified structure for destination folder YYYY/MM/DD/YYYYMMDDHHmmss\_\<md5sum\>.\<ext\>
- _deduplication_ - as long as time and content are the same, resulting name will be the same and (re)import of the file will be skipped
- _deletion from destination_ - files can be deleted on destination after import and it will not get reimported by default as long as same processing db is used. 

You can import multiple times from the same source (ie. non-wiped sd card with new photos) and it will only import photos not yet stored in destination before (no reimport of stored and manually deleted ones unless explicitly requested)

## Install

Download from releases and put in PATH

## Simplest use

    importgoblin import --from my/new/images --to ~/MyPhotos
