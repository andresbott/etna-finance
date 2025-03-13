<script setup>
import InputIcon from 'primevue/inputicon'
import InputText from 'primevue/inputtext'
import IconField from 'primevue/iconfield'
import {ref} from "vue";
import {useBookmarkStore} from "@/stores/bookmark.js";


const searchText = ref('');
const lastSearch = ref('');
const bkmStore = useBookmarkStore()

function handleEnter(event) {
  if (event.key === 'Enter') {
    search()
  }
  if (event.key === 'Escape') {
    cleanSearch()
  }
}
function handleKeyUp(){
  if (searchText.value === ""){
    search()
  }
}
function search(){
  if (lastSearch.value !== searchText.value){
    lastSearch.value = searchText.value
    bkmStore.Load(searchText.value);
  }
}

function cleanSearch(){
  searchText.value = ""
  search()
}
</script>

<template>
    <div class="searchbar">
        <IconField>
            <InputIcon class="pi pi-search cursor-pointer"
                       @click="search"
            />
          <InputText
              placeholder="Search"
              v-model="searchText"
              @keydown="handleEnter"
              @keyup="handleKeyUp"
          />
          <InputIcon class="pi pi-times cursor-pointer"
                     @click="cleanSearch"
          />
          </IconField>

    </div>
</template>

<style scoped>
.searchbar {
    width: 35rem;
}

.searchbar input {
    width: 100%;
}
</style>
