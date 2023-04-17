{% macro foo() %} exsisting {% endmacro %}

{%allowmissingval%} No tag {{bar()}}no error {%endallowmissingval%}

{%allowmissingval%} Finally calling an{{foo}}value. {%endallowmissingval%}

{%allowmissingval%} This will return an empty{{b}}string. {%endallowmissingval%}