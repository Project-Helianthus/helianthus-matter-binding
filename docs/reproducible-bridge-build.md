# Reproducible bridge build record

This repository records the bridge build. It does not build the gateway binary, and it does not add the job that produces the builds.

The bridge binary is the standalone gateway built without the nomatter profile. This repository builds no second executable.

The targets are linux/arm64, and linux/arm with GOARM=7. Each target is built twice from a clean state with trimpath, version-control stamping off and an empty build id. The two digests of one target must be equal.

Each recorded run carries the Go toolchain, the module checksum of this module, the module checksum of go-fabric, and an SBOM for that binary.

The official add-on build is not this job. A bench run cites the digest and is a separate session. The official add-on build stays without Matter.
