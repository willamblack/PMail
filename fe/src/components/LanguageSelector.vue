<template>
  <el-select
      :model-value="lang.lang"
      size="small"
      class="language-selector"
      :aria-label="lang.language"
      @change="setLanguage"
  >
    <el-option :label="lang.chinese" value="zhCn"/>
    <el-option :label="lang.english" value="en"/>
  </el-select>
</template>

<script setup>
import lang, {setLanguage} from "../i18n/i18n";
</script>

<style scoped>
.language-selector {
  width: 112px;
}
</style>
