---
name: "storekit-config"
description: "Troubleshoots StoreKit Configuration file issues in iOS apps. Invoke when StoreKit products can't load or subscription storekit files are missing from bundle."
---

# StoreKit Configuration Troubleshooting

This skill provides comprehensive steps for diagnosing and fixing StoreKit Configuration file issues in iOS applications, specifically for MoeMemos.

## Common Symptoms

- StoreKit products show "Product unavailable"
- Products loaded count: 0
- `Subscription.storekit NOT found in main bundle` in logs

## Step-by-Step Troubleshooting

### 1. Add Detailed Logging

Add comprehensive logging to verify:
- File existence in bundle
- File content
- JSON format validity
- Product parsing

```swift
// In StoreKitManager.swift
public func loadProducts() async {
    print("[StoreKitManager] [1/4] Checking if Subscription.storekit exists...")
    if let storeKitURL = Bundle.main.url(forResource: "Subscription", withExtension: "storekit") {
        print("[StoreKitManager] ✅ Found Subscription.storekit at: \(storeKitURL)")
        
        // Read and print file content
        print("[StoreKitManager] [2/4] Reading file content...")
        let fileContent = try String(contentsOf: storeKitURL, encoding: .utf8)
        print("[StoreKitManager] ✅ File content:")
        print(fileContent)
        
        // Parse and validate JSON
        print("[StoreKitManager] [3/4] Parsing JSON and checking format...")
        // Check version, products array, subscriptionGroups
    }
    
    print("[StoreKitManager] [4/4] Loading products from StoreKit...")
    let storeProducts = try await Product.products(for: productIDs)
    print("[StoreKitManager] ✅ Products loaded, count: \(storeProducts.count)")
}
```

### 2. Verify Xcode Project Configuration

**Critical Check:** Ensure the `.storekit` file is in the **Resources Build Phase**!

#### Check project.pbxproj:

1. Verify file reference exists:
```
A4F11FE02F64DFBC00FE1650 /* Subscription.storekit */ = {isa = PBXFileReference; ... };
```

2. Add PBXBuildFile entry:
```
A4F11FE12F64DFBC00FE1651 /* Subscription.storekit in Resources */ = {isa = PBXBuildFile; fileRef = A4F11FE02F64DFBC00FE1650 /* Subscription.storekit */; };
```

3. Add to Resources Build Phase:
```
files = (
    ...,
    A4F11FE12F64DFBC00FE1651 /* Subscription.storekit in Resources */,
);
```

### 3. Verify StoreKit Configuration File Content

Ensure the file has:

```json
{
  "identifier": "Subscription",
  "version": {
    "major": 4,
    "minor": 0
  },
  "subscriptionGroups": [
    {
      "id": "MemosVIP",
      "name": "Memos VIP",
      "subscriptions": [
        {
          "productID": "com.memos.vip.yearly",
          "type": "RecurringSubscription",
          "recurringSubscriptionPeriod": "P1Y",
          "localizations": [
            {
              "description": "解锁5GB存储空间、无限备忘录、无广告体验",
              "displayName": "Memos VIP 年度会员",
              "locale": "zh_CN"
            }
          ]
        }
      ]
    }
  ],
  "settings": {
    "_locale": "zh_CN",
    "_storefront": "CHN"
  }
}
```

**Key Fields:**
- ✅ `productID` must match your code
- ✅ `localizations` must have `displayName` and `description`
- ✅ `recurringSubscriptionPeriod`: P1Y (yearly), P1M (monthly)
- ✅ `_storefront`: CHN for China, USA for US
- ✅ `_locale`: zh_CN for Chinese, en_US for English

### 4. Xcode Scheme Configuration

Verify in `.xcscheme`:
```xml
<StoreKitConfigurationFileReference
   identifier = "MoeMemos/Subscription.storekit">
</StoreKitConfigurationFileReference>
```

## Clean and Rebuild

Always clean build after changes:

```bash
# In Xcode: Product → Clean Build Folder (Shift+Cmd+K)
# Or via command line:
xcodebuild clean -project MoeMemos.xcodeproj -scheme MoeMemos
```

## File Locations

StoreKit Configuration files should be in:
- Primary: `MoeMemos/Subscription.storekit` (added to target)
- Backup: Root directory or Packages/Subscription/

## Common Mistakes to Avoid

1. ❌ File exists in project but NOT in Resources Build Phase
2. ❌ `products` array is empty (subscriptions go in `subscriptionGroups`)
3. ❌ Missing `localizations` with `displayName`
4. ❌ Wrong `recurringSubscriptionPeriod` format
5. ❌ Forgetting to clean build after changes

## Verification Steps

After applying fixes:

1. Clean build folder
2. Build and run
3. Check logs for:
   - `✅ Found Subscription.storekit at:`
   - `✅ JSON parsed successfully`
   - `✅ Products loaded successfully, count: 1`
4. Verify subscription products appear in UI

This skill provides a complete workflow for troubleshooting StoreKit Configuration issues in iOS applications.
