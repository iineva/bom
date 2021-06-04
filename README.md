# bom

Apple iOS / macOS Assets.car decoder, write in golang.

[ipa-server](https://github.com/iineva/ipa-server) use this to decode app icons in `Assets.cart`

* BOM: Bill of Materials
* Asset Catalog: Assets.car, and It's a BOM file with special block

### Decode bom file

```golang
import "github.com/iineva/bom/pkg/bom"

fileName := "Assets.car"
f, _ := os.Open(fileName)
defer f.Close()
b := bom.New(f)
err := b.Parse() // parse header first

// read block names
names := b.BlockNames()
// read block
r, err := b.ReadBlock(names[0])
// read tree block
err := b.ReadTree("FACETKEYS", func(k io.Reader, d io.Reader) error {
    // handle tree block item
})
```

### Decode Asset Catalog

```golang
import "github.com/iineva/bom/pkg/asset"

fileName := "Assets.car"
f, _ := os.Open(fileName)
defer f.Close()
b, _ := asset.NewWithReadSeeker(f)
// read image with name
img, err := b.Image("AppIcon")
```

# Reference

<https://blog.timac.org/2018/1018-reverse-engineering-the-car-file-format/>
<https://blog.timac.org/2018/1112-quicklook-plugin-to-visualize-car-files/>
<https://github.com/hogliux/bomutils>
<http://lingyuncxb.com/2019/04/14/HumbleAssetCatalog/>
<https://github.com/lzfse/lzfse>
<https://github.com/iineva/go-lzfse>
