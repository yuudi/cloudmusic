<script lang="ts">
	import { activeLocale, setLanguage, t } from '#lib/i18n/index.js';
</script>

<label class="flex items-center gap-2 text-sm">
	<span class="sr-only">{$t('common.language')}</span>
	<select
		class="select w-auto"
		value={$activeLocale}
		onchange={(event) => setLanguage(event.currentTarget.value)}
	>
		<option value="en" lang="en">English</option>
		<option value="zh-CN" lang="zh-CN">简体中文</option>
	</select>
</label>
