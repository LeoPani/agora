"use client";

import { useEffect, useState } from "react";

const STORAGE_KEY = "agora_intro_seen";

export function IntroAnimation() {
  const [visible, setVisible]     = useState(false);
  const [fading, setFading]       = useState(false);
  const [showSkip, setShowSkip]   = useState(false);

  useEffect(() => {
    if (typeof window === "undefined") return;

    const prefersReduced = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
    if (prefersReduced) return;

    const seen = localStorage.getItem(STORAGE_KEY);
    if (seen) return;

    setVisible(true);

    const skipTimer  = setTimeout(() => setShowSkip(true), 1500);
    const doneTimer  = setTimeout(() => dismiss(), 7500);

    return () => {
      clearTimeout(skipTimer);
      clearTimeout(doneTimer);
    };
  }, []);

  function dismiss() {
    setFading(true);
    localStorage.setItem(STORAGE_KEY, "1");
    setTimeout(() => setVisible(false), 1000);
  }

  if (!visible) return null;

  return (
    <div
      className={`agora-intro${fading ? " fading" : ""}`}
      onClick={dismiss}
      role="dialog"
      aria-label="Apresentação do Ágora"
    >
      <svg
        viewBox="0 0 900 500"
        xmlns="http://www.w3.org/2000/svg"
        style={{ width: "min(90vw, 900px)", height: "auto" }}
      >
        <defs>
          <radialGradient id="eyeGlowIntro" cx="50%" cy="50%" r="50%">
            <stop offset="0%"   stopColor="#D4A017" stopOpacity="1"/>
            <stop offset="60%"  stopColor="#D4A017" stopOpacity="0.3"/>
            <stop offset="100%" stopColor="#D4A017" stopOpacity="0"/>
          </radialGradient>
          <radialGradient id="bgVignette" cx="50%" cy="50%" r="70%">
            <stop offset="0%"   stopColor="#1A1329" stopOpacity="0.4"/>
            <stop offset="100%" stopColor="#0d0a1a" stopOpacity="0"/>
          </radialGradient>
        </defs>

        {/* vignette */}
        <rect width="900" height="500" fill="url(#bgVignette)"/>

        {/* estrelas */}
        <g opacity="0.4">
          <circle cx="120" cy="80"  r="0.8" fill="#fff"/>
          <circle cx="780" cy="120" r="0.8" fill="#fff"/>
          <circle cx="650" cy="50"  r="0.5" fill="#fff"/>
          <circle cx="200" cy="160" r="0.6" fill="#fff"/>
          <circle cx="830" cy="200" r="0.7" fill="#fff"/>
          <circle cx="60"  cy="200" r="0.5" fill="#fff"/>
          <circle cx="450" cy="60"  r="0.6" fill="#fff"/>
        </g>

        {/* ondas de radar */}
        <g transform="translate(450, 250)">
          <circle r="40" fill="none" stroke="#6B21A8" strokeWidth="1.2" opacity="0">
            <animate attributeName="r"       from="40" to="220" dur="4.5s" begin="2s"   repeatCount="indefinite"/>
            <animate attributeName="opacity" from="0.5" to="0"  dur="4.5s" begin="2s"   repeatCount="indefinite"/>
          </circle>
          <circle r="40" fill="none" stroke="#6B21A8" strokeWidth="1.2" opacity="0">
            <animate attributeName="r"       from="40" to="220" dur="4.5s" begin="3.5s" repeatCount="indefinite"/>
            <animate attributeName="opacity" from="0.5" to="0"  dur="4.5s" begin="3.5s" repeatCount="indefinite"/>
          </circle>
          <circle r="40" fill="none" stroke="#6B21A8" strokeWidth="1.2" opacity="0">
            <animate attributeName="r"       from="40" to="220" dur="4.5s" begin="5s"   repeatCount="indefinite"/>
            <animate attributeName="opacity" from="0.5" to="0"  dur="4.5s" begin="5s"   repeatCount="indefinite"/>
          </circle>
        </g>

        {/* linha do horizonte */}
        <line x1="450" y1="280" x2="450" y2="280" stroke="#6B21A8" strokeWidth="1.5" opacity="0">
          <animate attributeName="opacity" from="0" to="1"   dur="0.8s" begin="0.2s" fill="freeze"/>
          <animate attributeName="x1"      from="450" to="220" dur="1.2s" begin="0.2s" fill="freeze"/>
          <animate attributeName="x2"      from="450" to="680" dur="1.2s" begin="0.2s" fill="freeze"/>
        </line>

        {/* OLHO DE ARGOS — formato amendoado pontudo */}
        <g transform="translate(450, 250)">
          {/* halo dourado */}
          <circle r="45" fill="url(#eyeGlowIntro)" opacity="0">
            <animate attributeName="opacity" from="0" to="0.5"  dur="1.2s" begin="2s"   fill="freeze"/>
            <animate attributeName="r"       values="45;55;45"  dur="4s"   begin="3.2s" repeatCount="indefinite"/>
          </circle>

          {/* corpo do olho amendoado */}
          <path d="M -45 0 Q -35 -22 0 -22 Q 35 -22 45 0 Q 35 22 0 22 Q -35 22 -45 0 Z"
                fill="#1A1329" stroke="#6B21A8" strokeWidth="2.5" opacity="0"
                transform="scale(0)">
            <animateTransform attributeName="transform" type="scale"
                              from="0" to="1" dur="1.2s" begin="1.5s" fill="freeze" additive="replace"/>
            <animate attributeName="opacity" from="0" to="1" dur="0.8s" begin="1.5s" fill="freeze"/>
          </path>

          {/* micro-movimento de pálpebra */}
          <path d="M -45 0 Q 0 -8 45 0" fill="none" stroke="#6B21A8" strokeWidth="0.5" opacity="0">
            <animate attributeName="opacity" from="0" to="0.4" dur="0.5s" begin="3s" fill="freeze"/>
          </path>

          {/* íris dourada */}
          <circle r="0" fill="#D4A017">
            <animate attributeName="r" from="0"  to="11"     dur="0.6s" begin="2.4s" fill="freeze"/>
            <animate attributeName="r" values="11;12;11"     dur="3s"   begin="3.5s" repeatCount="indefinite"/>
          </circle>

          {/* pupila */}
          <circle r="0" fill="#0d0a1a">
            <animate attributeName="r" from="0" to="5" dur="0.4s" begin="2.8s" fill="freeze"/>
          </circle>

          {/* catchlight */}
          <circle cx="-1.5" cy="-1.5" r="0" fill="#fff" opacity="0.7">
            <animate attributeName="r" from="0" to="1.2" dur="0.3s" begin="3.1s" fill="freeze"/>
          </circle>

          {/* micro-movimento do olhar */}
          <animateTransform attributeName="transform" type="translate"
                            values="0 0; 1.5 0; 0 0; -1 0.5; 0 0"
                            keyTimes="0; 0.3; 0.5; 0.7; 1"
                            dur="8s" begin="5s" repeatCount="indefinite"/>
        </g>

        {/* 4 COLUNAS */}
        <g opacity="0">
          <animate attributeName="opacity" from="0" to="1" dur="0.5s" begin="3.2s" fill="freeze"/>
          <rect x="290" y="280" width="14" height="0" fill="#6B21A8">
            <animate attributeName="height" from="0" to="85" dur="0.8s" begin="3.2s" fill="freeze"/>
            <animate attributeName="y"      from="280" to="195" dur="0.8s" begin="3.2s" fill="freeze"/>
          </rect>
          <rect x="286" y="280" width="22" height="0" fill="#6B21A8">
            <animate attributeName="height" from="0" to="5" dur="0.3s" begin="4s" fill="freeze"/>
            <animate attributeName="y"      from="280" to="190" dur="0.3s" begin="4s" fill="freeze"/>
          </rect>
        </g>
        <g opacity="0">
          <animate attributeName="opacity" from="0" to="1" dur="0.5s" begin="3.4s" fill="freeze"/>
          <rect x="340" y="280" width="14" height="0" fill="#6B21A8">
            <animate attributeName="height" from="0" to="78" dur="0.8s" begin="3.4s" fill="freeze"/>
            <animate attributeName="y"      from="280" to="202" dur="0.8s" begin="3.4s" fill="freeze"/>
          </rect>
          <rect x="336" y="280" width="22" height="0" fill="#6B21A8">
            <animate attributeName="height" from="0" to="5" dur="0.3s" begin="4.2s" fill="freeze"/>
            <animate attributeName="y"      from="280" to="197" dur="0.3s" begin="4.2s" fill="freeze"/>
          </rect>
        </g>
        <g opacity="0">
          <animate attributeName="opacity" from="0" to="1" dur="0.5s" begin="3.6s" fill="freeze"/>
          <rect x="546" y="280" width="14" height="0" fill="#6B21A8">
            <animate attributeName="height" from="0" to="78" dur="0.8s" begin="3.6s" fill="freeze"/>
            <animate attributeName="y"      from="280" to="202" dur="0.8s" begin="3.6s" fill="freeze"/>
          </rect>
          <rect x="542" y="280" width="22" height="0" fill="#6B21A8">
            <animate attributeName="height" from="0" to="5" dur="0.3s" begin="4.4s" fill="freeze"/>
            <animate attributeName="y"      from="280" to="197" dur="0.3s" begin="4.4s" fill="freeze"/>
          </rect>
        </g>
        <g opacity="0">
          <animate attributeName="opacity" from="0" to="1" dur="0.5s" begin="3.8s" fill="freeze"/>
          <rect x="596" y="280" width="14" height="0" fill="#6B21A8">
            <animate attributeName="height" from="0" to="85" dur="0.8s" begin="3.8s" fill="freeze"/>
            <animate attributeName="y"      from="280" to="195" dur="0.8s" begin="3.8s" fill="freeze"/>
          </rect>
          <rect x="592" y="280" width="22" height="0" fill="#6B21A8">
            <animate attributeName="height" from="0" to="5" dur="0.3s" begin="4.6s" fill="freeze"/>
            <animate attributeName="y"      from="280" to="190" dur="0.3s" begin="4.6s" fill="freeze"/>
          </rect>
        </g>

        {/* ÓRBITAS KEPLERIANAS */}
        <g transform="translate(450, 250)" opacity="0">
          <animate attributeName="opacity" from="0" to="0.18" dur="1.5s" begin="4.8s" fill="freeze"/>
          <ellipse cx="0" cy="0" rx="90"  ry="38" fill="none" stroke="#6B21A8" strokeWidth="0.5" transform="rotate(-15)"/>
          <ellipse cx="0" cy="0" rx="130" ry="50" fill="none" stroke="#6B21A8" strokeWidth="0.5" transform="rotate(20)"/>
          <ellipse cx="0" cy="0" rx="175" ry="60" fill="none" stroke="#6B21A8" strokeWidth="0.5" transform="rotate(-5)"/>
        </g>

        {/* PLANETAS */}
        <g transform="translate(450, 250)" opacity="0">
          <animate attributeName="opacity" from="0" to="1" dur="1s" begin="5s" fill="freeze"/>

          <g transform="rotate(-15)">
            <g><animateTransform attributeName="transform" type="rotate" from="0" to="360" dur="7s" repeatCount="indefinite"/>
              <ellipse cx="90" cy="0" rx="4" ry="4" fill="#D4A017"/>
            </g>
            <g><animateTransform attributeName="transform" type="rotate" from="160" to="520" dur="7s" repeatCount="indefinite"/>
              <circle cx="90" cy="0" r="2.2" fill="#D4A017" opacity="0.65"/>
            </g>
          </g>

          <g transform="rotate(20)">
            <g><animateTransform attributeName="transform" type="rotate" from="360" to="0" dur="13s" repeatCount="indefinite"/>
              <circle cx="130" cy="0" r="3.5" fill="#D4A017"/>
            </g>
            <g><animateTransform attributeName="transform" type="rotate" from="180" to="-180" dur="13s" repeatCount="indefinite"/>
              <circle cx="130" cy="0" r="2" fill="#D4A017" opacity="0.55"/>
            </g>
          </g>

          <g transform="rotate(-5)">
            <g><animateTransform attributeName="transform" type="rotate" from="0" to="360" dur="22s" repeatCount="indefinite"/>
              <circle cx="175" cy="0" r="3" fill="#D4A017" opacity="0.85"/>
            </g>
            <g><animateTransform attributeName="transform" type="rotate" from="200" to="560" dur="22s" repeatCount="indefinite"/>
              <circle cx="175" cy="0" r="1.8" fill="#D4A017" opacity="0.45"/>
            </g>
          </g>
        </g>

        {/* TEXTO */}
        <text x="450" y="400" textAnchor="middle"
              style={{ fontFamily: "-apple-system, sans-serif", fontSize: "46px", fontWeight: 300,
                       letterSpacing: "0.22em", fill: "#ffffff", opacity: 0 }}>
          ÁGORA
          <animate attributeName="opacity" from="0" to="1" dur="1.2s" begin="5s" fill="freeze"/>
        </text>

        <text x="450" y="430" textAnchor="middle"
              style={{ fontFamily: "-apple-system, sans-serif", fontSize: "14px", fontWeight: 400,
                       letterSpacing: "0.4em", fill: "#D4A017", opacity: 0 }}>
          BY ARGOS
          <animate attributeName="opacity" from="0" to="1" dur="1s" begin="5.8s" fill="freeze"/>
        </text>

        <text x="450" y="465" textAnchor="middle"
              style={{ fontFamily: "-apple-system, sans-serif", fontSize: "11px", fontWeight: 400,
                       letterSpacing: "0.3em", fill: "rgba(255,255,255,0.4)", opacity: 0 }}>
          RADAR DE INTELIGÊNCIA DE INOVAÇÃO
          <animate attributeName="opacity" from="0" to="1" dur="1s" begin="6.4s" fill="freeze"/>
        </text>
      </svg>

      {showSkip && (
        <button
          onClick={(e) => { e.stopPropagation(); dismiss(); }}
          style={{
            position: "absolute",
            bottom: "2rem",
            right: "2rem",
            background: "rgba(107,33,168,0.3)",
            border: "1px solid rgba(107,33,168,0.4)",
            color: "rgba(255,255,255,0.6)",
            padding: "8px 20px",
            borderRadius: "6px",
            fontSize: "12px",
            letterSpacing: "0.1em",
            cursor: "pointer",
            animation: "fadeIn 0.4s ease",
          }}
        >
          Pular →
        </button>
      )}
    </div>
  );
}
