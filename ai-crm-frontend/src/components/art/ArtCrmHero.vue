<template>
  <svg class="crm-hero" viewBox="0 0 960 560" role="img" aria-label="CRM hero">
    <defs>
      <linearGradient id="crm-bg" x1="0" y1="0" x2="1" y2="1">
        <stop offset="0%" stop-color="#f7f9ff" />
        <stop offset="100%" stop-color="#eaf0ff" />
      </linearGradient>
      <linearGradient id="crm-primary" x1="0" y1="0" x2="1" y2="1">
        <stop offset="0%" stop-color="#6d8eea" />
        <stop offset="100%" stop-color="#4b6fd4" />
      </linearGradient>
      <linearGradient id="crm-green" x1="0" y1="0" x2="1" y2="1">
        <stop offset="0%" stop-color="#49c78f" />
        <stop offset="100%" stop-color="#2da66f" />
      </linearGradient>
    </defs>

    <rect x="20" y="20" width="920" height="520" rx="28" fill="url(#crm-bg)" />

    <circle cx="130" cy="115" r="10" fill="#c5d4fa" />
    <circle cx="838" cy="108" r="14" fill="#c5d4fa" />
    <circle cx="195" cy="468" r="16" fill="#d4e0ff" />

    <rect x="92" y="92" width="276" height="178" rx="18" fill="#ffffff" stroke="#dae3fb" />
    <text x="116" y="126" fill="#6e7da6" font-size="18" font-weight="600">Customers</text>
    <circle cx="136" cy="172" r="21" fill="#e8efff" />
    <circle cx="136" cy="164" r="8" fill="#5c82e6" />
    <path d="M120 186c4-9 10-14 16-14s12 5 16 14" fill="#5c82e6" />
    <rect x="168" y="154" width="166" height="14" rx="7" fill="#e7ecfb" />
    <rect x="168" y="178" width="126" height="12" rx="6" fill="#edf1fb" />
    <rect x="168" y="199" width="92" height="12" rx="6" fill="#edf1fb" />
    <rect x="112" y="228" width="104" height="24" rx="12" fill="url(#crm-green)" />
    <text x="132" y="245" fill="#fff" font-size="13" font-weight="600">+ New Lead</text>

    <rect x="392" y="84" width="476" height="206" rx="18" fill="#ffffff" stroke="#dae3fb" />
    <text x="420" y="120" fill="#6e7da6" font-size="18" font-weight="600">Pipeline & Forecast</text>
    <rect x="420" y="146" width="84" height="20" rx="10" fill="#edf2ff" />
    <text x="438" y="160" fill="#6383df" font-size="12" font-weight="700">Q2</text>
    <rect x="514" y="146" width="98" height="20" rx="10" fill="#edf2ff" />
    <text x="533" y="160" fill="#6383df" font-size="12" font-weight="700">SMB</text>
    <rect x="622" y="146" width="112" height="20" rx="10" fill="#edf2ff" />
    <text x="645" y="160" fill="#6383df" font-size="12" font-weight="700">Enterprise</text>
    <path d="M434 236c36-31 70-44 101-39 35 5 58 35 90 34 38-2 53-44 88-53 34-8 74 8 125 47" fill="none" stroke="url(#crm-primary)" stroke-width="5" stroke-linecap="round" />
    <circle cx="434" cy="236" r="6" fill="#4f74d9" />
    <circle cx="535" cy="197" r="6" fill="#4f74d9" />
    <circle cx="625" cy="231" r="6" fill="#4f74d9" />
    <circle cx="713" cy="178" r="6" fill="#4f74d9" />
    <circle cx="838" cy="225" r="6" fill="#4f74d9" />
    <rect x="744" y="106" width="106" height="44" rx="12" fill="#eef4ff" />
    <text x="760" y="126" fill="#7383a9" font-size="12">Won This Month</text>
    <text x="760" y="143" fill="#3d63c9" font-size="16" font-weight="700">$1.92M</text>

    <rect x="92" y="296" width="372" height="190" rx="18" fill="#ffffff" stroke="#dae3fb" />
    <text x="116" y="332" fill="#6e7da6" font-size="18" font-weight="600">Funnel</text>
    <polygon points="146,358 398,358 362,388 182,388" fill="#dbe6ff" />
    <polygon points="168,394 376,394 344,424 200,424" fill="#bdd2ff" />
    <polygon points="190,430 354,430 326,460 218,460" fill="#8fb2ff" />
    <text x="404" y="381" fill="#7a87aa" font-size="13">MQL 1,240</text>
    <text x="404" y="417" fill="#6b7ea8" font-size="13">SQL 620</text>
    <text x="404" y="453" fill="#5d74a5" font-size="13">Won 186</text>

    <rect x="490" y="312" width="378" height="174" rx="18" fill="#ffffff" stroke="#dae3fb" />
    <text x="516" y="346" fill="#6e7da6" font-size="18" font-weight="600">Service & SLA</text>
    <rect x="520" y="366" width="150" height="44" rx="12" fill="#eef3ff" />
    <text x="536" y="384" fill="#7383a9" font-size="12">Open Tickets</text>
    <text x="536" y="402" fill="#365ec4" font-size="18" font-weight="700">46</text>
    <rect x="688" y="366" width="150" height="44" rx="12" fill="#eef3ff" />
    <text x="704" y="384" fill="#7383a9" font-size="12">SLA in 15 min</text>
    <text x="704" y="402" fill="#365ec4" font-size="18" font-weight="700">8</text>
    <rect x="520" y="424" width="318" height="30" rx="15" fill="#eef3ff" />
    <rect x="520" y="424" width="251" height="30" rx="15" fill="url(#crm-green)" />
    <text x="649" y="444" text-anchor="middle" fill="#fff" font-size="13" font-weight="700">SLA On-Time 79%</text>
  </svg>
</template>

<script setup>
</script>

<style scoped>
.crm-hero {
  width: min(760px, 92%);
  height: auto;
  filter: drop-shadow(0 22px 46px rgba(67, 102, 178, 0.14));
}
</style>
